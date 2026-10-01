// Package deployment contiene la lógica pura del gestor de modelos: construir
// la orden de despliegue, interpretar tópicos y ACKs y mapear estados. No
// depende de Postgres, Kafka ni MQTT, para poder probarse de forma aislada.
package deployment

import (
	"fmt"
	"regexp"
	"strings"

	"ceiot-tf-background/modules/model-deployment/models"
)

// Estados de DEVICE_MODEL.STATUS.
const (
	DBPending    = "pending"    // orden creada, sin ACK aún (se reenvía)
	DBDownloaded = "downloaded" // descargado y verificado, sin activar
	DBActive     = "active"     // 'current' del dispositivo apunta a esta versión
	DBFailed     = "failed"     // el dispositivo rechazó o no pudo desplegar
	DBInactive   = "inactive"   // estuvo activa y otra versión la reemplazó
	DBSuperseded = "superseded" // quedó pendiente y se pidió otra versión antes del ACK
)

// Mismas reglas que el model-manager del dispositivo: el nombre del modelo
// se usa como carpeta en la Pi, así que se valida también aquí para fallar en
// el servidor (con un log claro) y no en la Pi.
var (
	modelNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	sha256Re    = regexp.MustCompile(`^[A-Fa-f0-9]{64}$`)
)

// BuildCommand arma la orden MQTT para una versión de modelo.
// Si baseURL está vacío, la URL va relativa y el dispositivo la completa con
// su SERVER_HTTP_BASE.
func BuildCommand(mv models.ModelVersion, baseURL string) (models.DeployCommand, error) {
	if !modelNameRe.MatchString(mv.ModelName) || strings.Contains(mv.ModelName, "..") {
		return models.DeployCommand{}, fmt.Errorf(
			"MODEL.NAME %q no es válido como carpeta en el dispositivo (use letras, números, '.', '_' o '-')", mv.ModelName)
	}
	if mv.Version <= 0 {
		return models.DeployCommand{}, fmt.Errorf("versión inválida: %d", mv.Version)
	}
	if !sha256Re.MatchString(mv.SHA256) {
		return models.DeployCommand{}, fmt.Errorf("MODEL_VERSION %d sin SHA256 válido", mv.ID)
	}
	path := strings.TrimLeft(mv.PathTFLite, "/")
	if path == "" || strings.Contains(path, "..") {
		return models.DeployCommand{}, fmt.Errorf("MODEL_VERSION %d con PATH_TFLITE inválido: %q", mv.ID, mv.PathTFLite)
	}

	return models.DeployCommand{
		IDModel:        mv.ModelName,
		IDModelVersion: mv.ID,
		Version:        mv.Version,
		URL:            strings.TrimRight(baseURL, "/") + "/artifacts/" + path,
		SHA256:         strings.ToLower(mv.SHA256),
		Activate:       true,
	}, nil
}

// DeviceTopic reemplaza el marcador ___DEVICE___ del tópico de publicación.
func DeviceTopic(template, idDevice string) string {
	return strings.Replace(template, "___DEVICE___", idDevice, 1)
}

// ParseDeployTopic extrae el id del dispositivo de devices/<id>/deploy.
func ParseDeployTopic(topic string) (string, bool) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 || parts[0] != "devices" || parts[2] != "deploy" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

// MapAckStatus traduce el estado del ACK del dispositivo al de DEVICE_MODEL.
func MapAckStatus(s string) (string, bool) {
	switch s {
	case "activated":
		return DBActive, true
	case "downloaded":
		return DBDownloaded, true
	case "failed":
		return DBFailed, true
	}
	return "", false
}

// ValidateEvent revisa el evento Kafka y devuelve la lista de dispositivos
// sin duplicados ni vacíos.
func ValidateEvent(ev models.DeployEvent) ([]string, error) {
	if ev.IDModelVersion <= 0 {
		return nil, fmt.Errorf("id_model_version inválido: %d", ev.IDModelVersion)
	}
	seen := map[string]bool{}
	var devices []string
	for _, d := range ev.Devices {
		d = strings.TrimSpace(d)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		devices = append(devices, d)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("el evento no trae dispositivos")
	}
	return devices, nil
}
