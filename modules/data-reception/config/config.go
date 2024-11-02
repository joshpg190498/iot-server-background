package config

import (
	"fmt"
	"net/url"
	"os"

	"ceiot-tf-background/modules/data-reception/models"
)

func LoadEnvVars() (*models.Config, error) {
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	kafkaBrokers := []string{kafkaBroker}
	kafkaTopics := []string{"device-data-events"}

	mqttClientID := "background-data-reception-mqtt-client"
	mqttProtocol := os.Getenv("MQTT_PROTOCOL")
	mqttHost := os.Getenv("MQTT_HOST")
	mqttPort := os.Getenv("MQTT_PORT")
	mqttBroker := fmt.Sprintf("%s://%s:%s", mqttProtocol, mqttHost, mqttPort)

	mqttSubDataTopic := "devices/+/data"
	mqttSubTopics := []string{mqttSubDataTopic}

	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDB := os.Getenv("POSTGRES_DB")
	encodedPostgresPassword := url.QueryEscape(postgresPassword)
	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", postgresUser, encodedPostgresPassword, postgresHost, postgresPort, postgresDB)

	config := &models.Config{
		KafkaBrokers:  kafkaBrokers,
		KafkaTopics:   kafkaTopics,
		MQTTClientID:  mqttClientID,
		MQTTBroker:    mqttBroker,
		MQTTSubTopics: mqttSubTopics,
		PostgresURL:   postgresURL,
	}

	return config, nil
}
