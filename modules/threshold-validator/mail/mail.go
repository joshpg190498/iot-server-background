package mail

import (
	"ceiot-tf-background/modules/threshold-validator/models"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

func newSmtpClient(smtpConfig models.SmtpConfig) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%s", smtpConfig.Host, smtpConfig.Port)
	auth := smtp.PlainAuth("", smtpConfig.User, smtpConfig.Password, smtpConfig.Host)
	conn, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpConfig.Host,
	}

	if err := conn.StartTLS(tlsConfig); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to start TLS: %w", err)
	}

	if err := conn.Auth(auth); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}
	log.Println("Connected to SMTP Client")
	return conn, nil
}

func splitRecipients(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func SendNotification(smtpConfig models.SmtpConfig, body string) bool {
	toRecipients := splitRecipients(smtpConfig.To)
	if len(toRecipients) == 0 {
		log.Println("No hay destinatarios (SMTP_TO) configurados, no se envía la notificación")
		return false
	}
	ccRecipients := splitRecipients(smtpConfig.Cc)

	client, err := newSmtpClient(smtpConfig)
	if err != nil {
		log.Println(err)
		return false
	}
	defer client.Close()

	if err := client.Mail(smtpConfig.User); err != nil {
		log.Println(err)
		return false
	}

	for _, recipient := range toRecipients {
		if err := client.Rcpt(recipient); err != nil {
			log.Println(err)
			return false
		}
	}

	for _, recipient := range ccRecipients {
		if err := client.Rcpt(recipient); err != nil {
			log.Println(err)
			return false
		}
	}

	w, err := client.Data()
	if err != nil {
		log.Println(err)
		return false
	}

	subject := "Plataforma de monitoreo de rendimiento para computadoras de placa única basadas en Linux"
	to := strings.Join(toRecipients, ", ")
	var msg string
	if len(ccRecipients) > 0 {
		cc := strings.Join(ccRecipients, ", ")
		msg = fmt.Sprintf("Subject: %s\r\nTo: %s\r\nCc: %s\r\n\r\n%s", subject, to, cc, body)
	} else {
		msg = fmt.Sprintf("Subject: %s\r\nTo: %s\r\n\r\n%s", subject, to, body)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		log.Println("Error writing message:", err)
		return false
	}

	if err := w.Close(); err != nil {
		log.Println("Error closing Data writer:", err)
		return false
	}
	return true
}

func BuildContent(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData) (string, error) {
	if setting.ThresholdValue == nil {
		return "", errors.New("threshold value is nil")
	}

	functions := getBuildContentFunctions()
	function, exists := functions[dataPayload.Parameter]
	if !exists {
		return "", errors.New("function not found")
	}
	return function(dataPayload.IDDevice, *setting.ThresholdValue, exceededRegister), nil
}

func buildContentCPUTemperature(id_device string, threshold float64, exceededRegister models.ThresholdExceededData) string {
	emailContent := fmt.Sprintf(
		"El sensor %s en el dispositivo %s reportó %.2f °C, superando el umbral definido de %.2f °C",
		exceededRegister.Key, id_device, exceededRegister.Value, threshold)
	return emailContent
}

func buildContentDiskUsage(id_device string, threshold float64, exceededRegister models.ThresholdExceededData) string {
	emailContent := fmt.Sprintf(
		"El disco %s en el dispositivo %s reportó un uso del %.2f %%, superando el umbral definido de %.2f %%",
		exceededRegister.Key, id_device, exceededRegister.Value, threshold)
	return emailContent
}

func buildContentRAMUsage(id_device string, threshold float64, exceededRegister models.ThresholdExceededData) string {
	emailContent := fmt.Sprintf(
		"El dispositivo %s reportó un uso de RAM de %.2f %%, superando el umbral definido de %.2f %%",
		id_device, exceededRegister.Value, threshold)
	return emailContent
}

func buildContentCPUUsage(id_device string, threshold float64, exceededRegister models.ThresholdExceededData) string {
	emailContent := fmt.Sprintf(
		"El dispositivo %s reportó un uso de CPU de %.2f %%, superando el umbral definido de %.2f %%",
		id_device, exceededRegister.Value, threshold)
	return emailContent
}

func buildContentLoadAverage(id_device string, threshold float64, exceededRegister models.ThresholdExceededData) string {
	emailContent := fmt.Sprintf(
		"El dispositivo %s reportó una carga de CPU de %.2f %%, superando el umbral definido de %.2f %%",
		id_device, exceededRegister.Value, threshold)
	return emailContent
}

type FuncType func(id_device string, threshold float64, exceededRegister models.ThresholdExceededData) string

func getBuildContentFunctions() map[string]FuncType {
	return map[string]FuncType{
		"ram":          buildContentRAMUsage,
		"disk":         buildContentDiskUsage,
		"cpu_temp":     buildContentCPUTemperature,
		"cpu_usage":    buildContentCPUUsage,
		"load_average": buildContentLoadAverage,
	}
}
