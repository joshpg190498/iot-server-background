package config

import (
	"fmt"
	"os"

	"ceiot-tf-background/modules/data-processing/models"
)

func LoadEnvVars() (*models.Config, error) {
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
