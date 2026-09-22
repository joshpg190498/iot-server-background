package main

import (
	"ceiot-tf-background/internal/kafka"
	"ceiot-tf-background/internal/mqtt"
	"ceiot-tf-background/modules/device-configuration/config"
	"ceiot-tf-background/modules/device-configuration/models"
	"ceiot-tf-background/modules/device-configuration/postgres"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"encoding/json"
	"log"
)

var (
	cfg *models.Config
)

func main() {
	loadConfiguration()
	initializeDatabase()

	startMQTTClient()
	kafka.InitializeReader(cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopics, kafkaHandleMessage)

	go periodicDatabaseCheck()

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

func kafkaHandleMessage(topic string, message []byte) error {
	if len(cfg.KafkaTopics) == 0 || topic != cfg.KafkaTopics[0] {
		return nil
	}

	kafkaMessage, err := parseKafkaMessage(message)
	if err != nil {
		log.Printf("Mensaje Kafka no parseable, se descarta sin reintentar: %v", err)
		return nil
	}

	return publishConfigurationToDevice(kafkaMessage.IDDevice, kafkaMessage.HashUpdate, kafkaMessage.Type)
}

func publishConfigurationToDevice(idDevice string, hashUpdate string, idType string) error {
	deviceReadingSettings, err := postgres.GetDeviceReadingSettings(idDevice)
	if err != nil {
		return fmt.Errorf("error obteniendo device reading settings para %s: %w", idDevice, err)
	}
	messageConfigPayload := buildMessageConfigPayload(idDevice, hashUpdate, idType, deviceReadingSettings)

	mqttPayload, err := stringifyPayload(messageConfigPayload)
	if err != nil {
		return fmt.Errorf("error serializando config para %s: %w", idDevice, err)
	}

	mqttConfigDeviceTopic := strings.Replace(cfg.MQTTPubConfigTopicTemp, "___DEVICE___", messageConfigPayload.IDDevice, 1)

	if !mqtt.PublishData(mqttConfigDeviceTopic, mqttPayload) {
		return fmt.Errorf("error publicando configuración MQTT para %s", idDevice)
	}
	return nil
}

func parseKafkaMessage(message []byte) (models.KafkaMessage, error) {
	var kafkaMessage models.KafkaMessage
	if err := json.Unmarshal(message, &kafkaMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.KafkaMessage{}, err
	}
	return kafkaMessage, nil
}

func buildMessageConfigPayload(IDDevice string, HashUpdate string, Type string, deviceReadingSettings []models.DeviceReadingSetting) models.MessageConfigPayload {
	return models.MessageConfigPayload{
		IDDevice:   IDDevice,
		HashUpdate: HashUpdate,
		Type:       Type,
		Settings:   deviceReadingSettings,
	}
}

func mqttHandleMessage(topic string, message []byte) {
	topicDeviceID, ok := parseDeviceConfigTopic(topic)
	if !ok {
		return
	}

	responseConfigPayload, err := parseMqttMessage(message)
	if err != nil {
		return
	}

	if responseConfigPayload.IDDevice != topicDeviceID {
		log.Printf("Mensaje descartado: IDDevice del payload (%q) no coincide con el del tópico (%q)", responseConfigPayload.IDDevice, topicDeviceID)
		return
	}

	if err := postgres.UpdateDeviceAndInsertInfo(responseConfigPayload); err != nil {
		log.Printf("Error actualizando configuración/confirmación del dispositivo %s: %v", responseConfigPayload.IDDevice, err)
		return
	}
}

func parseDeviceConfigTopic(topic string) (deviceID string, ok bool) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 || parts[0] != "devices" || parts[2] != "config" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func parseMqttMessage(message []byte) (models.ResponseConfigPayload, error) {
	var mqttMessage models.ResponseConfigPayload
	if err := json.Unmarshal(message, &mqttMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.ResponseConfigPayload{}, err
	}
	return mqttMessage, nil
}

func stringifyPayload(payload any) (string, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error converting to JSON: %s", err)
		return "", err
	}
	return string(jsonData), nil
}

func periodicDatabaseCheck() {
	for {
		notUpdatedDevices, err := postgres.GetNotUpdatedDevices()
		if err != nil {
			log.Printf("Error fetching not updated devices: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, device := range notUpdatedDevices {
			if err := publishConfigurationToDevice(device.IDDevice, device.HashUpdate, device.Type); err != nil {
				log.Printf("Error republicando configuración pendiente para %s: %v", device.IDDevice, err)
			}
		}

		time.Sleep(1 * time.Minute)
	}
}
