package postgres

import (
	"context"
	"log"

	"ceiot-tf-background/go-modules/sbc-configuration/models"

	"github.com/jackc/pgx/v5"
)

var (
	db *pgx.Conn
)

func ConnectDB(connString string) error {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return err
	}

	db = conn
	log.Println("Connected to PostgreSQL")
	return nil
}

func CloseDB() {
	if db != nil {
		db.Close(context.Background())
		log.Println("PostgreSQL connection closed")
	}
}

func GetDeviceReadingSettings(idDevice string) ([]models.DeviceReadingSetting, error) {
	query := `
		SELECT ID_DEVICE, PARAMETER, PERIOD, ACTIVE
		FROM DEVICE_READING_SETTINGS
		WHERE ID_DEVICE = $1
	`

	rows, err := db.Query(context.Background(), query, idDevice)
	if err != nil {
		log.Println("Error getting device reading settings: %v", err)
		return nil, err
	}
	defer rows.Close()

	settings := []models.DeviceReadingSetting{}
	for rows.Next() {
		var setting models.DeviceReadingSetting
		if err := rows.Scan(&setting.IDDevice, &setting.Parameter, &setting.Period, &setting.Active); err != nil {
			log.Println("Error getting device reading settings: %v", err)
			return nil, err
		}
		settings = append(settings, setting)
	}

	return settings, nil
}

func UpdateSBCConfirmation(tx pgx.Tx, idDevice, hashUpdate, cfgType, updateDatetimeUTC string) error {
	query := `
		UPDATE DEVICE_UPDATES
		SET UPDATE_DATETIME_UTC = $1
		WHERE ID_DEVICE = $2 AND HASH_UPDATE = $3 AND TYPE = $4
	`

	_, err := tx.Exec(context.Background(), query, updateDatetimeUTC, idDevice, hashUpdate, cfgType)
	if err != nil {
		log.Printf("Error updating UPDATE_DATETIME_UTC: %v", err)
		return err
	}

	return nil
}

func InsertMainDeviceInformation(tx pgx.Tx, idDevice string, mainDeviceInfo models.MainDeviceInformation) error {
	query := `
		INSERT INTO MAIN_DEVICE_INFORMATION (
			ID_DEVICE, HOSTNAME, PROCESSOR, RAM, HOSTID, OS, KERNEL
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := tx.Exec(
		context.Background(),
		query,
		idDevice,
		mainDeviceInfo.Hostname,
		mainDeviceInfo.Processor,
		mainDeviceInfo.RAM,
		mainDeviceInfo.HostID,
		mainDeviceInfo.OS,
		mainDeviceInfo.Kernel,
	)
	if err != nil {
		log.Println("Error inserting main device information:", err)
		return err
	}
	return nil
}

func UpdateDeviceAndInsertInfo(rCfgPayload models.ResponseConfigPayload) error {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = UpdateSBCConfirmation(tx, rCfgPayload.IDDevice, rCfgPayload.HashUpdate, rCfgPayload.Type, rCfgPayload.UpdateDatetimeUTC)
	if err != nil {
		return err
	}

	if rCfgPayload.Type == "startup" {
		err = InsertMainDeviceInformation(tx, rCfgPayload.IDDevice, rCfgPayload.MainDeviceInformation)
		if err != nil {
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	log.Println("Transaction completed successfully for device configuration:", rCfgPayload.IDDevice)
	return nil
}
