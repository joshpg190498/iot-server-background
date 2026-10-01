// model-deployment: Gestor de modelos del servidor (Fig. 3.1 / 3.2 de la memoria).
//
// Flujo:
//  1. La API MLOps publica en Kafka (model-deploy-events):
//     {"id_model_version": N, "devices": ["rpi4-01", ...]}
//  2. Se registra cada despliegue en DEVICE_MODEL como 'pending' (la BD es la
//     fuente de verdad) y se publica la orden en server/deploy/<id_device>.
//  3. El model-manager del dispositivo descarga, verifica sha256, activa y
//     responde por devices/<id_device>/deploy → se actualiza DEVICE_MODEL.
//  4. Un chequeo periódico reenvía lo que siga 'pending' sin ACK (dispositivo
//     apagado, ACK perdido con QoS 0, etc.). El dispositivo es idempotente,
//     así que un reenvío nunca vuelve a descargar lo que ya tiene.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ceiot-tf-background/internal/kafka"
	"ceiot-tf-background/internal/mqtt"
	"ceiot-tf-background/modules/model-deployment/config"
	"ceiot-tf-background/modules/model-deployment/deployment"
	"ceiot-tf-background/modules/model-deployment/models"
	"ceiot-tf-background/modules/model-deployment/postgres"
)

var cfg *models.Config

func main() {
	var err error
	cfg, err = config.LoadEnvVars()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}
	if err := postgres.ConnectDB(cfg.PostgresURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	if cfg.ArtifactBaseURL == "" {
		log.Println("ARTIFACT_BASE_URL vacío: se enviarán URLs relativas (el dispositivo usa su SERVER_HTTP_BASE)")
	}

	go mqtt.ConnectClient(cfg.MQTTBroker, cfg.MQTTClientID, cfg.MQTTSubTopics, cfg.CertsDir, mqttHandleMessage)
	kafka.InitializeReader(cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopics, kafkaHandleMessage)

	go retryPendingLoop()

	waitForShutdown()
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("Señal de apagado recibida, cerrando conexiones...")
	kafka.Close()
	postgres.CloseDB()
	log.Println("Apagado completo.")
}

// kafkaHandleMessage: devolver error hace que Kafka NO confirme el offset y
// reintente. Solo se devuelve error ante fallas transitorias (BD caída); un
// evento inválido se descarta con log para no bloquear el consumidor.
func kafkaHandleMessage(topic string, message []byte) error {
	if topic != cfg.KafkaTopics[0] {
		return nil
	}
	ctx := context.Background()

	var ev models.DeployEvent
	if err := json.Unmarshal(message, &ev); err != nil {
		log.Printf("Evento de deploy no parseable, se descarta: %v", err)
		return nil
	}
	devices, err := deployment.ValidateEvent(ev)
	if err != nil {
		log.Printf("Evento de deploy inválido, se descarta: %v", err)
		return nil
	}

	mv, found, err := postgres.GetModelVersion(ctx, ev.IDModelVersion)
	if err != nil {
		return err // transitorio: reintentar
	}
	if !found {
		log.Printf("MODEL_VERSION %d no existe, se descarta el evento", ev.IDModelVersion)
		return nil
	}

	// Validar ANTES de crear filas: si la versión no es desplegable (sin
	// sha256, nombre inválido) no se deja nada 'pending' reintentándose.
	cmd, err := deployment.BuildCommand(mv, cfg.ArtifactBaseURL)
	if err != nil {
		log.Printf("MODEL_VERSION %d no desplegable, se descarta: %v", mv.ID, err)
		return nil
	}

	known, unknown, err := postgres.CreateDeployments(ctx, mv.ID, devices)
	if err != nil {
		return err
	}
	if len(unknown) > 0 {
		log.Printf("Dispositivos inexistentes en DEVICES, se omiten: %v", unknown)
	}

	for _, d := range known {
		// Si falla la publicación, la fila queda 'pending' y el reintento
		// periódico la vuelve a enviar: no hace falta devolver error aquí.
		publishCommand(d, cmd)
	}
	log.Printf("Deploy de %s v%d (MODEL_VERSION %d) enviado a %v", mv.ModelName, mv.Version, mv.ID, known)
	return nil
}

func publishCommand(idDevice string, cmd models.DeployCommand) bool {
	payload, err := json.Marshal(cmd)
	if err != nil {
		log.Printf("Error serializando orden de deploy: %v", err)
		return false
	}
	return mqtt.PublishData(deployment.DeviceTopic(cfg.MQTTPubDeployTopicTemp, idDevice), string(payload))
}

func mqttHandleMessage(topic string, message []byte) {
	topicDevice, ok := deployment.ParseDeployTopic(topic)
	if !ok {
		return
	}
	var ack models.DeployStatus
	if err := json.Unmarshal(message, &ack); err != nil {
		log.Printf("ACK de deploy no parseable (%s): %v", topic, err)
		return
	}
	if ack.IDDevice != topicDevice {
		log.Printf("ACK descartado: id_device del payload (%q) no coincide con el del tópico (%q)", ack.IDDevice, topicDevice)
		return
	}
	dbStatus, ok := deployment.MapAckStatus(ack.Status)
	if !ok {
		log.Printf("ACK con estado desconocido %q de %s, se ignora", ack.Status, ack.IDDevice)
		return
	}
	if ack.IDModelVersion <= 0 {
		// El dispositivo no pudo ni parsear la orden: no hay fila que actualizar.
		log.Printf("ACK %s de %s sin id_model_version: %s", ack.Status, ack.IDDevice, ack.Message)
		return
	}

	applied, err := postgres.ApplyAck(context.Background(), ack, dbStatus)
	switch {
	case err != nil:
		log.Printf("Error registrando ACK de %s (MODEL_VERSION %d): %v", ack.IDDevice, ack.IDModelVersion, err)
	case !applied:
		log.Printf("ACK de %s para MODEL_VERSION %d sin despliegue vigente (superseded o inexistente)", ack.IDDevice, ack.IDModelVersion)
	default:
		log.Printf("Deploy %s → %s en %s (%s v%d) %s", ack.Status, dbStatus, ack.IDDevice, ack.IDModel, ack.Version, ack.Message)
	}
}

func retryPendingLoop() {
	tick := cfg.RetryAfter / 2
	if tick < 5*time.Second {
		tick = 5 * time.Second
	}
	for {
		time.Sleep(tick)
		if !mqtt.IsConnected() {
			continue
		}
		ctx := context.Background()
		pending, err := postgres.GetPendingDeployments(ctx, cfg.RetryAfter)
		if err != nil {
			log.Printf("Error consultando despliegues pendientes: %v", err)
			continue
		}
		for _, p := range pending {
			cmd, err := deployment.BuildCommand(p.ModelVersion, cfg.ArtifactBaseURL)
			if err != nil {
				log.Printf("Despliegue pendiente no reenviable (%s, MODEL_VERSION %d): %v → failed", p.IDDevice, p.ID, err)
				_ = postgres.MarkFailed(ctx, p.IDDevice, p.ID)
				continue
			}
			if publishCommand(p.IDDevice, cmd) {
				log.Printf("Reenviado deploy sin ACK: %s ← %s v%d", p.IDDevice, p.ModelName, p.Version)
				if err := postgres.TouchDeployment(ctx, p.IDDevice, p.ID); err != nil {
					log.Printf("Error actualizando DEPLOYED_AT_UTC: %v", err)
				}
			}
		}
	}
}
