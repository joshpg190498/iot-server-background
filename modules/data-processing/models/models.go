package models

type Config struct {
	KafkaBrokers []string
	KafkaTopics  []string
	PostgresURL  string
}

type DeviceParameter struct {
	DeviceID string
	Param    string
}
