package main

import (
	"ceiot-tf-background/modules/threshold-validator/config"
	"ceiot-tf-background/modules/threshold-validator/evaluator"
	"ceiot-tf-background/modules/threshold-validator/mail"
	"ceiot-tf-background/modules/threshold-validator/models"
	"ceiot-tf-background/modules/threshold-validator/postgres"
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
	if topic != cfg.KafkaTopics[0] {
		return
	}

	dataPayload, err := parseKafkaMessage(message)
	if err != nil {
		return
	}

	if dataPayload.Data == nil {
		return
	}

	setting, err := postgres.GetParamater(dataPayload.IDDevice, dataPayload.Parameter)
	if err != nil {
		return
	}

	if !setting.HasThreshold {
		return
	}

	thresholdExceededData, err := evaluator.GetThresholdExceededData(setting, dataPayload)
	if err != nil {
		return
	}
	if len(thresholdExceededData) > 0 {
		sendNotificationsAndLog(dataPayload, setting, thresholdExceededData)
	} else {
		log.Printf("No exceeded data")
	}
}

func sendNotificationsAndLog(
	dataPayload models.DataPayload,
	setting models.DeviceReadingSetting,
	thresholdExceededData []models.ThresholdExceededData) {
	for _, exceededRegister := range thresholdExceededData {
		emailContent, err := mail.BuildContent(dataPayload, setting, exceededRegister)
		if err != nil {
			continue
		}
		sentEmail := mail.SendNotification(cfg.SmtpConfig, emailContent)
		err = postgres.InsertLog(dataPayload, setting, exceededRegister, sentEmail)
		if err != nil {
			log.Println(err)
		}
	}
}

func parseKafkaMessage(message []byte) (models.DataPayload, error) {
	var kafkaMessage models.DataPayload
	if err := json.Unmarshal(message, &kafkaMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.DataPayload{}, err
	}
	return kafkaMessage, nil
}
