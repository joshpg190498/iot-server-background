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
	ID                  int    `json:"id"`                    // Identificador de la actualización
	IDDevice            string `json:"id_device"`             // Identificador del dispositivo
	HashUpdate          string `json:"hash_update"`           // Identificador del mensaje
	Type                string `json:"type"`                  // Tipo de actualización
	CreationDatetimeUTC string `json:"creation_datetime_utc"` // Fecha de creación de la actualización
	UpdateDatetimeUTC   string `json:"update_datetime_utc"`   // Fecha de actualización del dispositivo
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

type NotUpdatedDevices struct {
	IDDevice   string
	HashUpdate string
	Type       string
}
