package models

type Config struct {
	KafkaBrokers  []string
	KafkaTopics   []string
	MQTTBroker    string
	MQTTClientID  string
	MQTTSubTopics []string
	PostgresURL   string
}

type DataPayload struct {
	IDDevice       string `json:"IDDevice"`
	Parameter      string `json:"Parameter"`
	Data           any    `json:"Data"`
	CollectedAtUtc string `json:"CollectedAtUtc"`
}

type MainDeviceInfo struct {
	HostID    string `json:"hostID"`
	Hostname  string `json:"hostname"`
	Kernel    string `json:"kernel"`
	OS        string `json:"os"`
	Processor string `json:"processor"`
	RAM       string `json:"ram"`
}

type RAMUsage struct {
	TotalRAM       int64   `json:"totalRAM"`
	FreeRAM        int64   `json:"freeRAM"`
	UsedRAM        int64   `json:"usedRAM"`
	UsedPercentRAM float64 `json:"usedPercentRAM"`
}

type DiskUsage struct {
	DiskName        string  `json:"diskName"`
	TotalDisk       int64   `json:"totalDisk"`
	FreeDisk        int64   `json:"freeDisk"`
	UsedDisk        int64   `json:"usedDisk"`
	UsedPercentDisk float64 `json:"usedPercentDisk"`
}

type NetworkStats struct {
	InterfaceName string `json:"interfaceName"`
	BytesSent     int64  `json:"bytesSent"`
	BytesRecv     int64  `json:"bytesRecv"`
	PacketsSent   int64  `json:"packetsSent"`
	PacketsRecv   int64  `json:"packetsRecv"`
	ErrOut        int64  `json:"errOut"`
	ErrIn         int64  `json:"errIn"`
	DropIn        int64  `json:"dropIn"`
	DropOut       int64  `json:"dropOut"`
}

type NetworkInformation struct {
	InterfaceName string   `json:"interfaceName"`
	MTU           int      `json:"mtu"`
	HardwareAddr  string   `json:"hardwareAddr"`
	Flags         []string `json:"flags"`
	Addrs         []string `json:"addrs"`
}

type CPUTemperature struct {
	SensorKey   string  `json:"sensorKey"`
	Temperature float64 `json:"temperature"`
}

type Uptime struct {
	UptimeMinutes int `json:"uptime"`
}

type LastReboot struct {
	LastReboot string `json:"lastReboot"`
}

type CPUUsage struct {
	CPUUsage float64 `json:"cpuUsage"`
}

type LoadAverage struct {
	LoadAverage1M  float64 `json:"loadAverage1m"`
	LoadAverage5M  float64 `json:"loadAverage5m"`
	LoadAverage15M float64 `json:"loadAverage15m"`
}
