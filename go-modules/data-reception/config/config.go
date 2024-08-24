package config

import (
	"fmt"
	"os"

	"ceiot-tf-background/go-modules/data-reception/models"
)

func LoadEnvVars() (*models.Config, error) {
	/* dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error al obtener el directorio actual: %w", err)
	}

	parentDir := filepath.Join(dir, "..", "..")
	envFile := filepath.Join(parentDir, ".env")

	err = godotenv.Load(envFile)
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}*/

	kafkaClientID := os.Getenv("SBC_DATA_RECEPTION_KAFKA_CLIENT_ID")
	kafkaGroupID := os.Getenv("SBC_DATA_RECEPTION_KAFKA_GROUP_ID")
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	kafkaBrokers := []string{kafkaBroker}
	kafkaTopicNewDeviceData := os.Getenv("KAFKA_TOPIC_NEW_DEVICE_DATA")
	kafkaTopicNewDeviceNotification := os.Getenv("KAFKA_TOPIC_NEW_DEVICE_NOTIFICATION")
	kafkaTopics := []string{kafkaTopicNewDeviceData, kafkaTopicNewDeviceNotification}

	mqttClientID := "mqtt-background-data-reception"
	mqttProtocol := os.Getenv("MQTT_PROTOCOL")
	mqttHost := os.Getenv("MQTT_HOST")
	mqttPort := os.Getenv("MQTT_PORT")
	mqttBroker := fmt.Sprintf("%s://%s:%s", mqttProtocol, mqttHost, mqttPort)

	mqttSubDataTopic := "DEVICES/+/DATA"
	mqttSubTopics := []string{mqttSubDataTopic}

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
		MQTTClientID:  mqttClientID,
		MQTTBroker:    mqttBroker,
		MQTTSubTopics: mqttSubTopics,
		PostgresURL:   postgresURL,
	}

	return config, nil
}
