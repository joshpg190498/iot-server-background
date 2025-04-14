package postgres

import (
	"ceiot-tf-background/modules/threshold-validator/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	db *pgxpool.Pool
)

func ConnectDB(connString string) error {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return err
	}

	db = pool
	log.Println("Connected to PostgreSQL")
	return nil
}

func CloseDB() {
	if db != nil {
		db.Close()
		log.Println("PostgreSQL connection closed")
	}
}

func GetParamater(id_device string, id_parameter string) (models.DeviceReadingSetting, error) {
	query := `
		SELECT DRS.ID_DEVICE, DRS.PARAMETER, DRS.PERIOD, 
			DRS.ACTIVE, DRS.THRESHOLD_VALUE, P.HAS_THRESHOLD, P.TABLE_POINTER 
		FROM DEVICE_READING_SETTINGS DRS
		INNER JOIN PARAMETERS P 
		ON DRS.PARAMETER = P.ID_PARAMETER
		WHERE DRS.ID_DEVICE = $1 AND P.ID_PARAMETER = $2
	`
	row := db.QueryRow(context.Background(), query, id_device, id_parameter)

	var setting models.DeviceReadingSetting
	err := row.Scan(&setting.IDDevice, &setting.Parameter, &setting.Period, &setting.Active,
		&setting.ThresholdValue, &setting.HasThreshold, &setting.TablePointer)
	if err != nil {
		log.Printf("Error getting device reading setting by parameter: %v", err)
		return models.DeviceReadingSetting{}, err
	}

	return setting, nil
}

func InsertLog(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData, sentEmail bool) error {
	ctx := context.Background()

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query, args, err := buildQueryToGetIdRef(dataPayload, setting, exceededRegister)
	if err != nil {
		return fmt.Errorf("error building query to get ID reference: %w", err)
	}

	var idRef int
	err = tx.QueryRow(ctx, query, args...).Scan(&idRef)
	if err != nil {
		return fmt.Errorf("error querying reference ID: %w", err)
	}

	dataJSON, err := json.Marshal(exceededRegister)
	if err != nil {
		return fmt.Errorf("error marshalling exceededRegister: %w", err)
	}

	insertQuery := `
		INSERT INTO THRESHOLD_ALERTS (ID_DEVICE, ID_PARAMETER, TABLE_POINTER, ID_REFERENCE, DATA, EMAIL_SENT)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, insertQuery, dataPayload.IDDevice, setting.Parameter, setting.TablePointer, idRef, string(dataJSON), sentEmail)
	if err != nil {
		return fmt.Errorf("error inserting into THRESHOLD_ALERTS: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	log.Println("Log inserted into THRESHOLD_ALERTS")
	return nil
}

func buildQueryToGetIdRef(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData) (string, []interface{}, error) {
	functions := getBuildQueryToGetIdRefFunctions()
	if function, exists := functions[dataPayload.Parameter]; exists {
		query, args := function(dataPayload, setting, exceededRegister)
		return query, args, nil
	} else {
		return "", nil, errors.New("function not found")
	}
}

func buildQueryToGetIdRefCPUTemperature(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData) (string, []interface{}) {
	query := `
		SELECT ID FROM %s WHERE ID_DEVICE = $1 AND SENSOR_KEY = $2 AND COLLECTED_AT_UTC = $3
	`
	query = fmt.Sprintf(query, setting.TablePointer)
	args := []interface{}{dataPayload.IDDevice, exceededRegister.Key, dataPayload.CollectedAtUtc}
	return query, args
}

func buildQueryToGetIdRefDiskUsage(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData) (string, []interface{}) {
	query := `
		SELECT ID FROM %s WHERE ID_DEVICE = $1 AND DISK_NAME = $2 AND COLLECTED_AT_UTC = $3
	`
	query = fmt.Sprintf(query, setting.TablePointer)
	args := []interface{}{dataPayload.IDDevice, exceededRegister.Key, dataPayload.CollectedAtUtc}
	return query, args
}

func buildQueryToGetIdRefRAMUsage(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData) (string, []interface{}) {
	query := `
		SELECT ID FROM %s WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC = $2
	`
	query = fmt.Sprintf(query, setting.TablePointer)
	args := []interface{}{dataPayload.IDDevice, dataPayload.CollectedAtUtc}
	return query, args
}

func buildQueryToGetIdRefCPUUsage(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData) (string, []interface{}) {
	query := `
		SELECT ID FROM %s WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC = $2
	`
	query = fmt.Sprintf(query, setting.TablePointer)
	args := []interface{}{dataPayload.IDDevice, dataPayload.CollectedAtUtc}
	return query, args
}

func buildQueryToGetIdRefLoadAverage(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData) (string, []interface{}) {
	query := `
		SELECT ID FROM %s WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC = $2
	`
	query = fmt.Sprintf(query, setting.TablePointer)
	args := []interface{}{dataPayload.IDDevice, dataPayload.CollectedAtUtc}
	return query, args
}

type FuncType func(dataPayload models.DataPayload, setting models.DeviceReadingSetting, exceededRegister models.ThresholdExceededData) (string, []interface{})

func getBuildQueryToGetIdRefFunctions() map[string]FuncType {
	return map[string]FuncType{
		"ram":          buildQueryToGetIdRefRAMUsage,
		"disk":         buildQueryToGetIdRefDiskUsage,
		"cpu_temp":     buildQueryToGetIdRefCPUTemperature,
		"cpu_usage":    buildQueryToGetIdRefCPUUsage,
		"load_average": buildQueryToGetIdRefLoadAverage,
	}
}

func ExistsRecentAlert(id_device, id_parameter, sinceUTC string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM THRESHOLD_ALERTS 
			WHERE ID_DEVICE = $1 
			AND ID_PARAMETER = $2 
			AND EMAIL_SENT = true 
			AND CREATED_AT_UTC >= $3
		)
	`
	var exists bool
	err := db.QueryRow(context.Background(), query, id_device, id_parameter, sinceUTC).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking recent alert existence: %w", err)
	}
	return exists, nil
}
