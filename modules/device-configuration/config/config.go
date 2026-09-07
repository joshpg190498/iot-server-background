package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"ceiot-tf-background/modules/device-configuration/models"
)

func LoadEnvVars() (*models.Config, error) {
	kafkaClientID := "background-device-configuration-kafka-client"
	kafkaGroupID := "device-update-events-handler-group"
	kafkaBrokers := splitAndTrim(os.Getenv("KAFKA_BROKER"), ",")
	kafkaTopics := []string{"device-update-events"}

	mqttClientID := "background-device-configuration-mqtt-client"
	mqttProtocol := os.Getenv("MQTT_PROTOCOL")
	mqttHost := os.Getenv("MQTT_HOST")
	mqttPort := os.Getenv("MQTT_PORT")
	mqttBroker := fmt.Sprintf("%s://%s:%s", mqttProtocol, mqttHost, mqttPort)

	mqttSubConfigTopic := "devices/+/config"
	mqttSubTopics := []string{mqttSubConfigTopic}

	mqttPubConfigTopicTemp := "server/config/___DEVICE___"

	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDB := os.Getenv("POSTGRES_DB")
	encodedPostgresUser := url.QueryEscape(postgresUser)
	encodedPostgresPassword := url.QueryEscape(postgresPassword)
	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", encodedPostgresUser, encodedPostgresPassword, postgresHost, postgresPort, postgresDB)

	config := &models.Config{
		KafkaClientID:          kafkaClientID,
		KafkaGroupID:           kafkaGroupID,
		KafkaBrokers:           kafkaBrokers,
		KafkaTopics:            kafkaTopics,
		MQTTClientID:           mqttClientID,
		MQTTBroker:             mqttBroker,
		MQTTSubTopics:          mqttSubTopics,
		MQTTPubConfigTopicTemp: mqttPubConfigTopicTemp,
		PostgresURL:            postgresURL,
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
