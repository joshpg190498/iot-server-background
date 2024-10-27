package config

import (
	"fmt"
	"os"
	"path/filepath"

	"ceiot-tf-background/modules/email-notification/models"

	"github.com/joho/godotenv"
)

func LoadEnvVars() (*models.Config, error) {
	dir, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("error al obtener el directorio actual: %w", err)
	}

	parentDir := filepath.Join(dir, "..", "..")
	envFile := filepath.Join(parentDir, ".env")

	err = godotenv.Load(envFile)
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	kafkaClientID := "background-email-notifications-kafka-client"
	kafkaGroupID := "device-data-events-notification-group"
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	kafkaBrokers := []string{kafkaBroker}
	kafkaTopics := []string{"device-data-events"}

	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDB := os.Getenv("POSTGRES_DB")
	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)

	config := &models.Config{
		KafkaClientID: kafkaClientID,
		KafkaGroupID:  kafkaGroupID,
		KafkaBrokers:  kafkaBrokers,
		KafkaTopics:   kafkaTopics,
		PostgresURL:   postgresURL,
	}

	return config, nil
}
