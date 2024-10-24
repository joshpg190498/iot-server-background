package models

type Config struct {
	KafkaClientID          string
	KafkaGroupID           string
	KafkaBrokers           []string
	KafkaTopics            []string
	MQTTBroker             string
	MQTTClientID           string
	MQTTSubTopics          []string
	PostgresURL            string
	MQTTPubConfigTopicTemp string
}

type KafkaMessage struct {
	ID                  int    `json:"id"`
	IDDevice            string `json:"id_device"`
	HashUpdate          string `json:"hash_update"`
	Type                string `json:"type"`
	CreationDatetimeUTC string `json:"creation_datetime_utc"`
	UpdateDatetimeUTC   string `json:"update_datetime_utc"`
}

type DeviceReadingSetting struct {
	IDDevice  string
	Parameter string
	Period    int
	Active    bool
}

type MessageConfigPayload struct {
	IDDevice   string
	HashUpdate string
	Type       string
	Settings   []DeviceReadingSetting
}

type MainDeviceInformation struct {
	HostID    string `json:"hostID"`
	Hostname  string `json:"hostname"`
	Kernel    string `json:"kernel"`
	OS        string `json:"os"`
	Processor string `json:"processor"`
	RAM       string `json:"ram"`
	CpuCount  int    `json:"cpuCount"`
}

type ResponseConfigPayload struct {
	IDDevice              string
	HashUpdate            string
	Type                  string
	MainDeviceInformation MainDeviceInformation
	UpdateDatetimeUTC     string
}
