package main

import (
	"ceiot-tf-background/go-modules/data-reception/config"
	"ceiot-tf-background/go-modules/data-reception/models"
	"ceiot-tf-background/go-modules/data-reception/postgres"
	"ceiot-tf-background/go-modules/utils/kafka"
	"ceiot-tf-background/go-modules/utils/mqtt"
	"encoding/json"
	"log"
	"strings"
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
	go kafka.InitializeWriter(cfg.KafkaBrokers)
}

func initializeDatabase() {
	err = postgres.ConnectDB(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
}

func mqttHandleMessage(topic string, message []byte) {
	if !strings.HasPrefix(topic, "DEVICES/") || !strings.HasSuffix(topic, "/DATA") {
		return
	}
	dataPayload, err := parseMqttMessage(message)
	if err != nil {
		return
	}
	log.Println(dataPayload)

	err = postgres.InsertData(dataPayload)
	if err != nil {
		log.Printf("Error inserting data: %v", err)
		return
	}

	kafka.PublishData(cfg.KafkaTopics[0], nil, message)
}

func parseMqttMessage(message []byte) (models.DataPayload, error) {
	var mqttMessage models.DataPayload
	if err := json.Unmarshal(message, &mqttMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.DataPayload{}, err
	}
	return mqttMessage, nil
}
