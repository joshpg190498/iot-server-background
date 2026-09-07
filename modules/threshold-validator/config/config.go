package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"ceiot-tf-background/modules/threshold-validator/models"
)

func LoadEnvVars() (*models.Config, error) {
	kafkaClientID := "background-threshold-validators-kafka-client"
	kafkaGroupID := "device-data-events-notification-group"
	kafkaBrokers := splitAndTrim(os.Getenv("KAFKA_BROKER"), ",")
	kafkaTopics := []string{"device-data-events"}

	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDB := os.Getenv("POSTGRES_DB")
	encodedPostgresUser := url.QueryEscape(postgresUser)
	encodedPostgresPassword := url.QueryEscape(postgresPassword)
	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", encodedPostgresUser, encodedPostgresPassword, postgresHost, postgresPort, postgresDB)

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

func splitAndTrim(value string, sep string) []string {
	var result []string
	for _, part := range strings.Split(value, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
