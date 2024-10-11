package config

import (
	"fmt"
	"os"
	"path/filepath"

	"ceiot-tf-background/go-modules/data-processing/models"

	"github.com/joho/godotenv"
)

func LoadEnvVars() (*models.Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error al obtener el directorio actual: %w", err)
	}

	parentDir := filepath.Join(dir, "..", "..")
	envFile := filepath.Join(parentDir, ".env")

	err = godotenv.Load(envFile)
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	kafkaBroker := os.Getenv("KAFKA_BROKER")
	kafkaBrokers := []string{kafkaBroker}
	kafkaTopicNewDeviceProcessedData := os.Getenv("KAFKA_TOPIC_NEW_DEVICE_PROCESSED_DATA")
	kafkaTopics := []string{kafkaTopicNewDeviceProcessedData}

	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDB := os.Getenv("POSTGRES_DB")
	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)

	config := &models.Config{
		KafkaBrokers: kafkaBrokers,
		KafkaTopics:  kafkaTopics,
		PostgresURL:  postgresURL,
	}

	return config, nil
}
