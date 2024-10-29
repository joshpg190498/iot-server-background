package models

type Config struct {
	KafkaClientID string
	KafkaGroupID  string
	KafkaBrokers  []string
	KafkaTopics   []string
	PostgresURL   string
	SmtpConfig    SmtpConfig
	SmtpTo        string
	SmtpCc        string
}

type DataPayload struct {
	IDDevice       string `json:"IDDevice"`
	Parameter      string `json:"Parameter"`
	Data           any    `json:"Data"`
	CollectedAtUtc string `json:"CollectedAtUtc"`
}

type DeviceReadingSetting struct {
	IDDevice       string
	Parameter      string
	Period         int
	Active         bool
	ThresholdValue *float64
	HasThreshold   bool
	TablePointer   string
}

type ThresholdExceededData struct {
	Key   string
	Value float64
}

type SmtpConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	To       string
	Cc       string
}
