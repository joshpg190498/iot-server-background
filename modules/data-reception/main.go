package main

import (
	"ceiot-tf-background/internal/kafka"
	"ceiot-tf-background/internal/mqtt"
	"ceiot-tf-background/modules/data-reception/config"
	"ceiot-tf-background/modules/data-reception/models"
	"ceiot-tf-background/modules/data-reception/postgres"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	cfg *models.Config
)

func main() {
	loadConfiguration()
	initializeDatabase()

	kafka.InitializeWriter(cfg.KafkaBrokers)
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

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Señal de apagado recibida, cerrando conexión a base de datos...")
	postgres.CloseDB()
	log.Println("Apagado completo.")
}

func mqttHandleMessage(topic string, message []byte) {
	topicDeviceID, ok := parseDeviceDataTopic(topic)
	if !ok {
		return
	}

	dataPayload, err := parseMqttMessage(message)
	if err != nil {
		return
	}

	if dataPayload.IDDevice != topicDeviceID {
		log.Printf("Mensaje descartado: IDDevice del payload (%q) no coincide con el del tópico (%q)", dataPayload.IDDevice, topicDeviceID)
		return
	}

	if err := postgres.InsertData(dataPayload); err != nil {
		log.Printf("Error inserting data for device %s, parameter %s: %v", dataPayload.IDDevice, dataPayload.Parameter, err)
		return
	}

	if !kafka.PublishData(cfg.KafkaTopics[0], []byte(dataPayload.IDDevice), message) {
		log.Printf("Error publicando en Kafka el evento de device %s, parameter %s (no se reintenta en este punto)", dataPayload.IDDevice, dataPayload.Parameter)
	}
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
		log.Printf("Error parsing kafka message: %v", err)
		return models.DataPayload{}, err
	}
	return mqttMessage, nil
}
