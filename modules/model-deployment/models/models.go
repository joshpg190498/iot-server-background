package models

import "time"

type Config struct {
	KafkaClientID          string
	KafkaGroupID           string
	KafkaBrokers           []string
	KafkaTopics            []string
	MQTTBroker             string
	MQTTClientID           string
	MQTTSubTopics          []string
	MQTTPubDeployTopicTemp string
	PostgresURL            string
	CertsDir               string
	ArtifactBaseURL        string        // ej. http://192.168.1.41:8000 ; vacío → URL relativa
	RetryAfter             time.Duration // reenvío de despliegues 'pending' sin ACK
}

// DeployEvent llega por Kafka (tópico model-deploy-events). Lo emite la API
// MLOps (Sprint 5); mientras tanto se produce a mano desde test/producer.kafka.
type DeployEvent struct {
	IDModelVersion int      `json:"id_model_version"`
	Devices        []string `json:"devices"`
}

// ModelVersion es lo que el gestor necesita de MODEL + MODEL_VERSION para
// construir la orden de despliegue.
type ModelVersion struct {
	ID         int
	ModelName  string
	Version    int
	PathTFLite string // relativo a ARTIFACTS_DIR: <name>/<version>/model.tflite
	SHA256     string
}

// PendingDeployment es una fila de DEVICE_MODEL en 'pending' a (re)enviar.
type PendingDeployment struct {
	IDDevice string
	ModelVersion
}

// DeployCommand baja por server/deploy/<id_device>. Debe coincidir con
// models.DeployCommand del model-manager del dispositivo.
type DeployCommand struct {
	IDModel        string `json:"id_model"`
	IDModelVersion int    `json:"id_model_version"`
	Version        int    `json:"version"`
	URL            string `json:"url"`
	SHA256         string `json:"sha256"`
	Activate       bool   `json:"activate"`
}

// DeployStatus sube por devices/<id_device>/deploy. Debe coincidir con
// models.DeployStatus del model-manager del dispositivo.
type DeployStatus struct {
	IDDevice       string `json:"id_device"`
	IDModel        string `json:"id_model"`
	IDModelVersion int    `json:"id_model_version"`
	Version        int    `json:"version"`
	Status         string `json:"status"`
	SHA256OK       bool   `json:"sha256_ok"`
	Message        string `json:"message"`
	TimestampUTC   string `json:"timestamp_utc"`
}
