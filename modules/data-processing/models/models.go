package models

type Config struct {
	PostgresURL string
}

type DeviceParameter struct {
	DeviceID string
	Param    string
}
