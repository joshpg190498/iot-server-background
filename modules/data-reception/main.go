package main

import (
	"ceiot-tf-background/internal/kafka"
	"ceiot-tf-background/internal/mqtt"
	"ceiot-tf-background/modules/data-reception/config"
	"ceiot-tf-background/modules/data-reception/models"
	"ceiot-tf-background/modules/data-reception/postgres"
	"encoding/json"
	"hash/fnv"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Message processing model:
//   - The MQTT callback runs in a single goroutine (paho). It must return fast,
//     so it only puts the message in a queue.
//   - numWorkers workers save each message in Postgres and send it to Kafka.
//   - All messages of one device go to the same worker (hash of the device id):
//     the order per device is kept, and different devices run in parallel.
//   - Each queue has a fixed size. If a queue is full, the callback waits
//     (backpressure) instead of using unlimited memory.
const (
	numWorkers   = 8
	queueSize    = 1000
	statsEvery   = 60 * time.Second
	drainTimeout = 30 * time.Second
)

type job struct {
	deviceID string
	payload  []byte
}

var (
	cfg *models.Config

	queues    []chan job
	workersWG sync.WaitGroup

	// stopping is set on shutdown. The lock makes sure no callback sends to a
	// queue after it is closed.
	stopMu   sync.RWMutex
	stopping bool

	// Counters for the periodic stats log
	received     atomic.Uint64
	saved        atomic.Uint64
	insertErrors atomic.Uint64
	discarded    atomic.Uint64
)

func main() {
	loadConfiguration()
	initializeDatabase()

	kafka.InitializeWriter(cfg.KafkaBrokers)
	startWorkers()
	go logStats()
	startMQTTClient()

	waitForShutdown()
}

func loadConfiguration() {
	var err error
	cfg, err = config.LoadEnvVars()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}
}

func startMQTTClient() {
	go mqtt.ConnectClient(cfg.MQTTBroker, cfg.MQTTClientID, cfg.MQTTSubTopics, cfg.CertsDir, mqttHandleMessage)
}

func initializeDatabase() {
	if err := postgres.ConnectDB(cfg.PostgresURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
}

func startWorkers() {
	queues = make([]chan job, numWorkers)
	for i := range queues {
		queues[i] = make(chan job, queueSize)
		workersWG.Add(1)
		go worker(queues[i])
	}
	log.Printf("Recepción iniciada: %d workers, cola de %d mensajes por worker", numWorkers, queueSize)
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Señal de apagado recibida, procesando los mensajes pendientes...")

	// Stop accepting new messages and close the queues
	stopMu.Lock()
	stopping = true
	for _, q := range queues {
		close(q)
	}
	stopMu.Unlock()

	// Wait for the workers to finish the queued messages (with a time limit)
	done := make(chan struct{})
	go func() {
		workersWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		log.Println("Mensajes pendientes procesados.")
	case <-time.After(drainTimeout):
		log.Printf("Tiempo de espera agotado (%s): quedaron mensajes sin procesar.", drainTimeout)
	}

	// Close flushes the messages still queued in the async Kafka writer
	kafka.Close()
	postgres.CloseDB()
	logStatsLine()
	log.Println("Apagado completo.")
}

// mqttHandleMessage runs in the paho goroutine: it only routes the message to
// the queue of its device and returns.
func mqttHandleMessage(topic string, message []byte) {
	topicDeviceID, ok := parseDeviceDataTopic(topic)
	if !ok {
		return
	}
	received.Add(1)

	// Copy the payload: the worker uses it after this function returns
	payload := append([]byte(nil), message...)

	stopMu.RLock()
	defer stopMu.RUnlock()
	if stopping {
		discarded.Add(1)
		return
	}
	queues[queueIndex(topicDeviceID)] <- job{deviceID: topicDeviceID, payload: payload}
}

func queueIndex(deviceID string) int {
	h := fnv.New32a()
	h.Write([]byte(deviceID))
	return int(h.Sum32() % uint32(len(queues)))
}

func worker(q <-chan job) {
	defer workersWG.Done()
	for j := range q {
		processMessage(j)
	}
}

func processMessage(j job) {
	dataPayload, err := parseMqttMessage(j.payload)
	if err != nil {
		discarded.Add(1)
		return
	}

	if dataPayload.IDDevice != j.deviceID {
		discarded.Add(1)
		log.Printf("Mensaje descartado: IDDevice del payload (%q) no coincide con el del tópico (%q)", dataPayload.IDDevice, j.deviceID)
		return
	}

	if err := postgres.InsertData(dataPayload); err != nil {
		insertErrors.Add(1)
		log.Printf("Error inserting data for device %s, parameter %s: %v", dataPayload.IDDevice, dataPayload.Parameter, err)
		return
	}
	saved.Add(1)

	// Async: only queues the event. Delivery errors are counted by the kafka package.
	if !kafka.PublishData(cfg.KafkaTopics[0], []byte(dataPayload.IDDevice), j.payload) {
		log.Printf("Error publicando en Kafka el evento de device %s, parameter %s (no se reintenta en este punto)", dataPayload.IDDevice, dataPayload.Parameter)
	}
}

// logStats writes one line per minute: messages received and saved, errors and
// the current queue length. It is the quick way to see if reception keeps up.
func logStats() {
	ticker := time.NewTicker(statsEvery)
	defer ticker.Stop()
	for range ticker.C {
		logStatsLine()
	}
}

func logStatsLine() {
	pending := 0
	for _, q := range queues {
		pending += len(q)
	}
	log.Printf("[stats] recibidos=%d guardados=%d errores_insert=%d descartados=%d errores_kafka=%d en_cola=%d",
		received.Load(), saved.Load(), insertErrors.Load(), discarded.Load(), kafka.AsyncErrors(), pending)
}

func parseDeviceDataTopic(topic string) (deviceID string, ok bool) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 || parts[0] != "devices" || parts[2] != "data" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func parseMqttMessage(message []byte) (models.DataPayload, error) {
	var mqttMessage models.DataPayload
	if err := json.Unmarshal(message, &mqttMessage); err != nil {
		log.Printf("Error parsing mqtt message: %v", err)
		return models.DataPayload{}, err
	}
	return mqttMessage, nil
}
