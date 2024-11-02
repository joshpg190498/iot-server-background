package config

import (
	"fmt"
	"net/url"
	"os"

	"ceiot-tf-background/modules/threshold-validator/models"
)

func LoadEnvVars() (*models.Config, error) {
	kafkaClientID := "background-threshold-validators-kafka-client"
	kafkaGroupID := "device-data-events-notification-group"
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	kafkaBrokers := []string{kafkaBroker}
	kafkaTopics := []string{"device-data-events"}

	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDB := os.Getenv("POSTGRES_DB")
	encodedPostgresPassword := url.QueryEscape(postgresPassword)
	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", postgresUser, encodedPostgresPassword, postgresHost, postgresPort, postgresDB)

	smtpConfig := models.SmtpConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		User:     os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASSWORD"),
		To:       os.Getenv("SMTP_TO"),
		Cc:       os.Getenv("SMTP_CC"),
	}

	config := &models.Config{
		KafkaClientID: kafkaClientID,
		KafkaGroupID:  kafkaGroupID,
		KafkaBrokers:  kafkaBrokers,
		KafkaTopics:   kafkaTopics,
		PostgresURL:   postgresURL,
		SmtpConfig:    smtpConfig,
	}

	return config, nil
}
