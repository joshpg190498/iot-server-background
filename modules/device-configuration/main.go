package main

import (
	"ceiot-tf-background/modules/device-configuration/config"
	"ceiot-tf-background/modules/device-configuration/models"
	"ceiot-tf-background/modules/device-configuration/postgres"
	"ceiot-tf-background/modules/utils/kafka"
	"ceiot-tf-background/modules/utils/mqtt"
	"strings"
	"time"

	"encoding/json"
	"log"
)

var (
	err error
	cfg *models.Config
)

func main() {
	loadConfiguration()
	startMQTTClient()
	startKafkaClient()
	initializeDatabase()
	go periodicDatabaseCheck()
	defer postgres.CloseDB()
	select {}
}

func loadConfiguration() {
	cfg, err = config.LoadEnvVars()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}
}

func startMQTTClient() {
	go mqtt.ConnectClient(cfg.MQTTBroker, cfg.MQTTClientID, cfg.MQTTSubTopics, mqttHandleMessage)
}

func startKafkaClient() {
	go kafka.InitializeReader(cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopics, kafkaHandleMessage)
}

func initializeDatabase() {
	err = postgres.ConnectDB(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
}

func kafkaHandleMessage(topic string, message []byte) {
	if topic != cfg.KafkaTopics[0] {
		return
	}

	kafkaMessage, err := parseKafkaMessage(message)
	if err != nil {
		return
	}

	publishConfigurationToDevice(kafkaMessage.IDDevice, kafkaMessage.HashUpdate, kafkaMessage.Type)
}

func publishConfigurationToDevice(idDevice string, hashUpdate string, idType string) {
	deviceReadingSettings, err := postgres.GetDeviceReadingSettings(idDevice)
	if err != nil {
		return
	}
	messageConfigPayload := buildMessageConfigPayload(idDevice, hashUpdate, idType, deviceReadingSettings)

	mqttPayload, err := stringifyPayload(messageConfigPayload)
	if err != nil {
		return
	}

	mqttConfigDeviceTopic := strings.Replace(cfg.MQTTPubConfigTopicTemp, "___DEVICE___", messageConfigPayload.IDDevice, 1)

	mqtt.PublishData(mqttConfigDeviceTopic, mqttPayload)
}

func parseKafkaMessage(message []byte) (models.KafkaMessage, error) {
	var kafkaMessage models.KafkaMessage
	if err := json.Unmarshal(message, &kafkaMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.KafkaMessage{}, err
	}
	return kafkaMessage, nil
}

func buildMessageConfigPayload(IDDevice string, HashUpdate string, Type string, deviceReadingSettings []models.DeviceReadingSetting) models.MessageConfigPayload {
	return models.MessageConfigPayload{
		IDDevice:   IDDevice,
		HashUpdate: HashUpdate,
		Type:       Type,
		Settings:   deviceReadingSettings,
	}
}

func mqttHandleMessage(topic string, message []byte) {
	if !strings.HasPrefix(topic, "devices/") || !strings.HasSuffix(topic, "/config") {
		return
	}

	responseConfigPayload, err := parseMqttMessage(message)
	if err != nil {
		return
	}

	err = postgres.UpdateDeviceAndInsertInfo(responseConfigPayload)
	if err != nil {
		return
	}
}

func parseMqttMessage(message []byte) (models.ResponseConfigPayload, error) {
	var mqttMessage models.ResponseConfigPayload
	if err := json.Unmarshal(message, &mqttMessage); err != nil {
		log.Printf("Error parsing kafka message: %v", err)
		return models.ResponseConfigPayload{}, err
	}
	return mqttMessage, nil
}

func stringifyPayload(payload any) (string, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error converting to JSON: %s", err)
		return "", nil
	}
	stringJsonData := string(jsonData)
	return stringJsonData, nil
}

func periodicDatabaseCheck() {
	for {
		notUpdatedDevices, err := postgres.GetNotUpdatedDevices()
		if err != nil {
			log.Printf("Error fetching not updated devices: %v", err)
			timer := time.NewTimer(1 * time.Minute)
			<-timer.C
			continue
		}

		for _, device := range notUpdatedDevices {
			publishConfigurationToDevice(device.IDDevice, device.HashUpdate, device.Type)
		}

		timer := time.NewTimer(1 * time.Minute)
		<-timer.C
	}
}
