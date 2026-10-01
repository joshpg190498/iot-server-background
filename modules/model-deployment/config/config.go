package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"ceiot-tf-background/modules/model-deployment/models"
)

const defaultRetrySec = 60

func LoadEnvVars() (*models.Config, error) {
	kafkaClientID := "background-model-deployment-kafka-client"
	kafkaGroupID := "model-deploy-events-handler-group"
	kafkaBrokers := splitAndTrim(os.Getenv("KAFKA_BROKER"), ",")
	kafkaTopics := []string{"model-deploy-events"}

	mqttClientID := "background-model-deployment-mqtt-client"
	mqttProtocol := os.Getenv("MQTT_PROTOCOL")
	mqttHost := os.Getenv("MQTT_HOST")
	mqttPort := os.Getenv("MQTT_PORT")
	mqttBroker := fmt.Sprintf("%s://%s:%s", mqttProtocol, mqttHost, mqttPort)

	mqttSubTopics := []string{"devices/+/deploy"}
	mqttPubDeployTopicTemp := "server/deploy/___DEVICE___"

	postgresURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		url.QueryEscape(os.Getenv("POSTGRES_USER")),
		url.QueryEscape(os.Getenv("POSTGRES_PASSWORD")),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)

	retrySec := defaultRetrySec
	if v := os.Getenv("DEPLOY_RETRY_SEC"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("DEPLOY_RETRY_SEC inválido: %q", v)
		}
		retrySec = n
	}

	return &models.Config{
		KafkaClientID:          kafkaClientID,
		KafkaGroupID:           kafkaGroupID,
		KafkaBrokers:           kafkaBrokers,
		KafkaTopics:            kafkaTopics,
		MQTTClientID:           mqttClientID,
		MQTTBroker:             mqttBroker,
		MQTTSubTopics:          mqttSubTopics,
		MQTTPubDeployTopicTemp: mqttPubDeployTopicTemp,
		PostgresURL:            postgresURL,
		CertsDir:               os.Getenv("CERTS_DIR"),
		ArtifactBaseURL:        strings.TrimRight(os.Getenv("ARTIFACT_BASE_URL"), "/"),
		RetryAfter:             time.Duration(retrySec) * time.Second,
	}, nil
}

func splitAndTrim(value string, sep string) []string {
	var result []string
	for _, part := range strings.Split(value, sep) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
