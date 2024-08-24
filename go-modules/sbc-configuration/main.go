package main

import (
	"ceiot-tf-background/go-modules/sbc-configuration/config"
	"ceiot-tf-background/go-modules/sbc-configuration/models"
	"ceiot-tf-background/go-modules/sbc-configuration/postgres"
	"ceiot-tf-background/go-modules/utils/kafka"
	"ceiot-tf-background/go-modules/utils/mqtt"
	"strings"

	"encoding/json"
	"log"
)

var (
	err error
	cfg *models.Config
)

func main() {
	loadConfiguration()
	startMQTTClient()
	startKafkaClient()
	initializeDatabase()
	defer postgres.CloseDB()
	select {}
}

func loadConfiguration() {
	cfg, err = config.LoadEnvVars()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}
}

func startMQTTClient() {
	go mqtt.ConnectClient(cfg.MQTTBroker, cfg.MQTTClientID, cfg.MQTTSubTopics, mqttHandleMessage)
}

func startKafkaClient() {
	go kafka.InitializeReader(cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopics, kafkaHandleMessage)
}

func initializeDatabase() {
	err = postgres.ConnectDB(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
}

func kafkaHandleMessage(topic string, message []byte) {
	kafkaMessage, err := parseKafkaMessage(message)
	if err != nil {
		return
	}
	deviceReadingSettings, err := postgres.GetDeviceReadingSettings(kafkaMessage.IDDevice)
	if err != nil {
		return
	}
	messageConfigPayload := buildMessageConfigPayload(kafkaMessage, deviceReadingSettings)

	mqttPayload, err := stringifyPayload(messageConfigPayload)
	if err != nil {
		return
	}

	mqttConfigDeviceTopic := strings.Replace(cfg.MQTTPubConfigTopicTemp, "___DEVICE___", messageConfigPayload.IDDevice, 1)

	mqtt.PublishData(mqttConfigDeviceTopic, mqttPayload)
}

func parseKafkaMessage(message []byte) (models.KafkaMessage, error) {
	var kafkaMessage models.KafkaMessage
	if err := json.Unmarshal(message, &kafkaMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.KafkaMessage{}, err
	}
	return kafkaMessage, nil
}

func buildMessageConfigPayload(kafkaMessage models.KafkaMessage, deviceReadingSettings []models.DeviceReadingSetting) models.MessageConfigPayload {
	return models.MessageConfigPayload{
		IDDevice:   kafkaMessage.IDDevice,
		HashUpdate: kafkaMessage.HashUpdate,
		Type:       kafkaMessage.Type,
		Settings:   deviceReadingSettings,
	}
}

func mqttHandleMessage(topic string, message []byte) {
	if !strings.HasPrefix(topic, "DEVICES/") || !strings.HasSuffix(topic, "/CONFIG") {
		return
	}

	responseConfigPayload, err := parseMqttMessage(message)
	if err != nil {
		return
	}

	err = postgres.UpdateDeviceAndInsertInfo(responseConfigPayload)
	if err != nil {
		return
	}
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
		return "", nil
	}
	stringJsonData := string(jsonData)
	return stringJsonData, nil
}
