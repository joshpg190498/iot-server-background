package models

type Config struct {
	KafkaClientID string
	KafkaGroupID  string
	KafkaBrokers  []string
	KafkaTopics   []string
	PostgresURL   string
}

type KafkaMessage struct {
	IDDevice       string `json:"IDDevice"`
	Parameter      string `json:"Parameter"`
	Data           any    `json:"Data"`
	CollectedAtUtc string `json:"CollectedAtUtc"`
}
