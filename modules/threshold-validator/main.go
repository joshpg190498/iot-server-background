package main

import (
	"ceiot-tf-background/internal/kafka"
	"ceiot-tf-background/modules/threshold-validator/config"
	"ceiot-tf-background/modules/threshold-validator/evaluator"
	"ceiot-tf-background/modules/threshold-validator/mail"
	"ceiot-tf-background/modules/threshold-validator/models"
	"ceiot-tf-background/modules/threshold-validator/postgres"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	cfg *models.Config
)

func main() {
	loadConfiguration()
	initializeDatabase()
	kafka.InitializeReader(cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopics, kafkaHandleMessage)
	waitForShutdown()
}

func loadConfiguration() {
	var err error
	cfg, err = config.LoadEnvVars()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}
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

	dataPayload, err := parseKafkaMessage(message)
	if err != nil {
		log.Printf("Mensaje Kafka no parseable, se descarta sin reintentar: %v", err)
		return nil
	}

	if dataPayload.Data == nil {
		return nil
	}

	setting, err := postgres.GetParamater(dataPayload.IDDevice, dataPayload.Parameter)
	if err != nil {
		if errors.Is(err, postgres.ErrSettingNotFound) {
			return nil
		}
		return err
	}

	if !setting.HasThreshold {
		return nil
	}

	if setting.ThresholdValue == nil {
		log.Printf("Inconsistencia de datos: %s/%s tiene HasThreshold=true pero ThresholdValue es NULL. Se omite sin reintentar.", dataPayload.IDDevice, dataPayload.Parameter)
		return nil
	}

	thresholdExceededData, err := evaluator.GetThresholdExceededData(setting, dataPayload)
	if err != nil {
		log.Printf("Error evaluando umbral para %s/%s (dato descartado, no se reintenta): %v", dataPayload.IDDevice, dataPayload.Parameter, err)
		return nil
	}

	if len(thresholdExceededData) == 0 {
		return nil
	}

	return sendNotificationsAndLog(dataPayload, setting, thresholdExceededData)
}

func sendNotificationsAndLog(
	dataPayload models.DataPayload,
	setting models.DeviceReadingSetting,
	thresholdExceededData []models.ThresholdExceededData) error {

	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	var lastErr error

	for _, exceededRegister := range thresholdExceededData {
		exists, err := postgres.ExistsRecentAlert(dataPayload.IDDevice, dataPayload.Parameter, exceededRegister.Key, oneHourAgo)
		if err != nil {
			log.Printf("Error checking recent alert for %s/%s/%s: %v", dataPayload.IDDevice, dataPayload.Parameter, exceededRegister.Key, err)
			lastErr = err
			continue
		}
		if exists {
			log.Printf("Cooldown activo para %s/%s/%s, se omite notificación", dataPayload.IDDevice, dataPayload.Parameter, exceededRegister.Key)
			continue
		}

		emailContent, err := mail.BuildContent(dataPayload, setting, exceededRegister)
		if err != nil {
			log.Printf("Error building email content for %s/%s/%s: %v", dataPayload.IDDevice, dataPayload.Parameter, exceededRegister.Key, err)
			continue
		}

		sentEmail := mail.SendNotification(cfg.SmtpConfig, emailContent)
		if err := postgres.InsertLog(dataPayload, setting, exceededRegister, sentEmail); err != nil {
			log.Println(err)
			lastErr = err
		}
	}

	return lastErr
}

func parseKafkaMessage(message []byte) (models.DataPayload, error) {
	var kafkaMessage models.DataPayload
	if err := json.Unmarshal(message, &kafkaMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.DataPayload{}, err
	}
	return kafkaMessage, nil
}
