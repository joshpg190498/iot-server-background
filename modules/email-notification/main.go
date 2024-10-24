package main

import (
	"ceiot-tf-background/modules/email-notification/config"
	"ceiot-tf-background/modules/email-notification/models"
	"ceiot-tf-background/modules/email-notification/postgres"
	"ceiot-tf-background/modules/utils/kafka"

	"encoding/json"
	"log"
)

var (
	err error
	cfg *models.Config
)

func main() {
	loadConfiguration()
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

	jsonData, err := json.Marshal(kafkaMessage)
	if err != nil {
		log.Printf("Error converting to JSON: %s", err)
	}
	stringJsonData := string(jsonData)

	log.Printf(stringJsonData)
}

func parseKafkaMessage(message []byte) (models.KafkaMessage, error) {
	var kafkaMessage models.KafkaMessage
	if err := json.Unmarshal(message, &kafkaMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.KafkaMessage{}, err
	}
	return kafkaMessage, nil
}
