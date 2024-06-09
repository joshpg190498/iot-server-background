package models

type Device struct {
	DeviceID    string
	Description string
}

type Config struct {
	MQTTHost         string
	MQTTPort         string
	MQTTClientID     string
	MQTTBroker       string
	MQTTSubTopics    []string
	DatabaseUsername string
	DatabasePassword string
	DatabaseDb       string
	DatabaseHost     string
}

type MessageDataPayload struct {
	DeviceID          string
	Parameter         string
	Data              any
	UpdateDatetimeUTC string
}
