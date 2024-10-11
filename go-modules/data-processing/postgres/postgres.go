package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
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

func GetActiveDevices() ([]string, error) {
	rows, err := db.Query(context.Background(), "SELECT ID_DEVICE FROM DEVICES WHERE ACTIVE = TRUE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []string
	for rows.Next() {
		var deviceID string
		if err := rows.Scan(&deviceID); err != nil {
			return nil, err
		}
		devices = append(devices, deviceID)
	}

	return devices, nil
}

func GetActiveParameters() ([]string, error) {
	rows, err := db.Query(context.Background(), "SELECT ID_PARAMETER FROM PARAMETERS WHERE HAS_THRESHOLD = TRUE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parameters []string
	for rows.Next() {
		var paramID string
		if err := rows.Scan(&paramID); err != nil {
			return nil, err
		}
		parameters = append(parameters, paramID)
	}

	return parameters, nil
}

func ProcessParameterData(deviceID string, param string) (int, error) {
	lastProcessed, err := GetLastProcessedTime(deviceID, param)
	if err != nil {
		return -1, fmt.Errorf("could not fetch last processed time: %w", err)
	}

	firstDataTime, err := GetFirstDataTimestamp(param, deviceID, lastProcessed)
	if err != nil {
		return -1, fmt.Errorf("could not fetch first data timestamp: %w", err)
	}

	if firstDataTime.IsZero() {
		log.Printf("No data available for processing for device %s and parameter %s", deviceID, param)
		return 0, nil
	}

	startTime := getStartTime(firstDataTime)

	currentUTC := time.Now().UTC().Truncate(time.Hour)

	if startTime == currentUTC {
		log.Printf("Start time (%v) is equal to current time (%v), skipping processing for device %s and parameter %s", startTime, currentUTC, deviceID, param)
		return 0, nil
	}

	fmt.Print(startTime, deviceID, param)

	switch param {
	case "cpu_temp":
		return processCPUTempData(deviceID, startTime)
	case "load_average":
		return processLoadAverageData(deviceID, startTime)
	case "disk":
		return processDiskUsageData(deviceID, startTime)
	case "ram":
		return processRAMUsageData(deviceID, startTime)
	case "cpu_usage":
		return processCPUUsageData(deviceID, startTime)
	default:
		return -1, fmt.Errorf("unsupported parameter: %s", param)
	}
}

func getStartTime(lastProcessed time.Time) time.Time {
	startTime := lastProcessed.Truncate(time.Hour)
	return startTime
}

func GetLastProcessedTime(deviceID, param string) (time.Time, error) {
	var lastProcessed time.Time
	err := db.QueryRow(context.Background(), `
		SELECT LAST_PROCESSED_AT
		FROM PROCESSING_POINTERS_HOURLY
		WHERE ID_DEVICE = $1 AND ID_PARAMETER = $2
	`, deviceID, param).Scan(&lastProcessed)

	if err != nil {
		return time.Time{}, nil
	}

	return lastProcessed, err
}

func GetFirstDataTimestamp(param string, deviceID string, lastProcessed time.Time) (time.Time, error) {
	var firstTimestamp *time.Time
	var query string
	switch param {
	case "cpu_temp":
		query = `
		SELECT MIN(COLLECTED_AT_UTC) 
			FROM CPU_TEMPERATURE 
		WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC >= $2`
	case "load_average":
		query = `
		SELECT MIN(COLLECTED_AT_UTC) 
			FROM LOAD_AVERAGE 
		WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC >= $2`
	case "disk":
		query = `
		SELECT MIN(COLLECTED_AT_UTC) 
			FROM DISK_USAGE 
		WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC >= $2`
	case "ram":
		query = `
		SELECT MIN(COLLECTED_AT_UTC) 
			FROM RAM_USAGE 
		WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC >= $2`
	case "cpu_usage":
		query = `
		SELECT MIN(COLLECTED_AT_UTC) 
			FROM CPU_USAGE
		WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC >= $2`
	default:
		return time.Time{}, fmt.Errorf("unsupported parameter: %s", param)
	}

	err := db.QueryRow(context.Background(), query, deviceID, lastProcessed).Scan(&firstTimestamp)

	if err != nil {
		if err == pgx.ErrNoRows {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}

	if firstTimestamp == nil {
		return time.Time{}, fmt.Errorf("No recent data for device %s and parameter %s", deviceID, param)
	}

	return *firstTimestamp, err
}

func processCPUTempData(deviceID string, startTime time.Time) (int, error) {
	nextHour := startTime.Add(time.Hour)
	endTime := nextHour.Add(-1 * time.Second)

	query := `
			INSERT INTO CPU_TEMPERATURE_HOURLY (
					ID_DEVICE, 
					SENSOR_KEY, 
					START_TIME, 
					AVG_TEMPERATURE, 
					MIN_TEMPERATURE, 
					MAX_TEMPERATURE, 
					ROW_COUNT, 
					INSERTED_AT_UTC
			)
			SELECT 
					ID_DEVICE, 
					SENSOR_KEY, 
					$2 AS START_TIME,
					AVG(TEMPERATURE) AS AVG_TEMPERATURE,
					MIN(TEMPERATURE) AS MIN_TEMPERATURE,
					MAX(TEMPERATURE) AS MAX_TEMPERATURE,
					COUNT(*) AS ROW_COUNT, 
					CURRENT_TIMESTAMP AS INSERTED_AT_UTC
			FROM CPU_TEMPERATURE
			WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC BETWEEN $2 AND $3
			GROUP BY ID_DEVICE, SENSOR_KEY
			ON CONFLICT (ID_DEVICE, SENSOR_KEY, START_TIME)
			DO UPDATE SET 
					AVG_TEMPERATURE = EXCLUDED.AVG_TEMPERATURE,
					MIN_TEMPERATURE = EXCLUDED.MIN_TEMPERATURE,
					MAX_TEMPERATURE = EXCLUDED.MAX_TEMPERATURE,
					ROW_COUNT = EXCLUDED.ROW_COUNT, 
					INSERTED_AT_UTC = EXCLUDED.INSERTED_AT_UTC
	`

	_, err := db.Exec(context.Background(), query, deviceID, startTime, endTime)
	if err != nil {
		return -1, fmt.Errorf("could not process CPU temperature data for device %s: %w", deviceID, err)
	}

	err = UpdateProcessingPointer(deviceID, "cpu_temp", nextHour)
	if err != nil {
		return -1, fmt.Errorf("could not update processing pointer for device %s and parameter cpu_temp: %w", deviceID, err)
	}

	return 1, nil
}

func processLoadAverageData(deviceID string, startTime time.Time) (int, error) {
	nextHour := startTime.Add(time.Hour)
	endTime := nextHour.Add(-1 * time.Second)

	query := `
			INSERT INTO LOAD_AVERAGE_HOURLY (
					ID_DEVICE,
					START_TIME,
					AVG_LOAD_AVERAGE_1M,
					MIN_LOAD_AVERAGE_1M,
					MAX_LOAD_AVERAGE_1M,
					AVG_LOAD_AVERAGE_5M,
					MIN_LOAD_AVERAGE_5M,
					MAX_LOAD_AVERAGE_5M,
					AVG_LOAD_AVERAGE_15M,
					MIN_LOAD_AVERAGE_15M,
					MAX_LOAD_AVERAGE_15M,
					ROW_COUNT,
					INSERTED_AT_UTC
			)
			SELECT 
					ID_DEVICE,
					$2 AS START_TIME,
					AVG(LOAD_AVERAGE_1M) AS AVG_LOAD_AVERAGE_1M,
					MIN(LOAD_AVERAGE_1M) AS MIN_LOAD_AVERAGE_1M,
					MAX(LOAD_AVERAGE_1M) AS MAX_LOAD_AVERAGE_1M,
					AVG(LOAD_AVERAGE_5M) AS AVG_LOAD_AVERAGE_5M,
					MIN(LOAD_AVERAGE_5M) AS MIN_LOAD_AVERAGE_5M,
					MAX(LOAD_AVERAGE_5M) AS MAX_LOAD_AVERAGE_5M,
					AVG(LOAD_AVERAGE_15M) AS AVG_LOAD_AVERAGE_15M,
					MIN(LOAD_AVERAGE_15M) AS MIN_LOAD_AVERAGE_15M,
					MAX(LOAD_AVERAGE_15M) AS MAX_LOAD_AVERAGE_15M,
					COUNT(*) AS ROW_COUNT,
					CURRENT_TIMESTAMP AS INSERTED_AT_UTC
			FROM LOAD_AVERAGE
			WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC BETWEEN $2 AND $3
			GROUP BY ID_DEVICE
			ON CONFLICT (ID_DEVICE, START_TIME)
			DO UPDATE SET 
					AVG_LOAD_AVERAGE_1M = EXCLUDED.AVG_LOAD_AVERAGE_1M,
					MIN_LOAD_AVERAGE_1M = EXCLUDED.MIN_LOAD_AVERAGE_1M,
					MAX_LOAD_AVERAGE_1M = EXCLUDED.MAX_LOAD_AVERAGE_1M,
					AVG_LOAD_AVERAGE_5M = EXCLUDED.AVG_LOAD_AVERAGE_5M,
					MIN_LOAD_AVERAGE_5M = EXCLUDED.MIN_LOAD_AVERAGE_5M,
					MAX_LOAD_AVERAGE_5M = EXCLUDED.MAX_LOAD_AVERAGE_5M,
					AVG_LOAD_AVERAGE_15M = EXCLUDED.AVG_LOAD_AVERAGE_15M,
					MIN_LOAD_AVERAGE_15M = EXCLUDED.MIN_LOAD_AVERAGE_15M,
					MAX_LOAD_AVERAGE_15M = EXCLUDED.MAX_LOAD_AVERAGE_15M,
					ROW_COUNT = EXCLUDED.ROW_COUNT,
					INSERTED_AT_UTC = EXCLUDED.INSERTED_AT_UTC
	`

	_, err := db.Exec(context.Background(), query, deviceID, startTime, endTime)
	if err != nil {
		return -1, fmt.Errorf("could not process load average data for device %s: %w", deviceID, err)
	}

	err = UpdateProcessingPointer(deviceID, "load_average", nextHour)
	if err != nil {
		return -1, fmt.Errorf("could not update processing pointer for device %s and parameter load_average: %w", deviceID, err)
	}

	return 1, nil
}

func processCPUUsageData(deviceID string, startTime time.Time) (int, error) {
	nextHour := startTime.Add(time.Hour)
	endTime := nextHour.Add(-1 * time.Second)

	query := `
			INSERT INTO CPU_USAGE_HOURLY (
					ID_DEVICE,
					START_TIME,
					AVG_CPU_USAGE,
					MIN_CPU_USAGE,
					MAX_CPU_USAGE,
					ROW_COUNT,
					INSERTED_AT_UTC
			)
			SELECT 
					ID_DEVICE,
					$2 AS START_TIME,
					AVG(CPU_USAGE) AS AVG_CPU_USAGE,
					MIN(CPU_USAGE) AS MIN_CPU_USAGE,
					MAX(CPU_USAGE) AS MAX_CPU_USAGE,
					COUNT(*) AS ROW_COUNT,
					CURRENT_TIMESTAMP AS INSERTED_AT_UTC
			FROM CPU_USAGE
			WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC BETWEEN $2 AND $3
			GROUP BY ID_DEVICE
			ON CONFLICT (ID_DEVICE, START_TIME)
			DO UPDATE SET 
					AVG_CPU_USAGE = EXCLUDED.AVG_CPU_USAGE,
					MIN_CPU_USAGE = EXCLUDED.MIN_CPU_USAGE,
					MAX_CPU_USAGE = EXCLUDED.MAX_CPU_USAGE,
					ROW_COUNT = EXCLUDED.ROW_COUNT,
					INSERTED_AT_UTC = EXCLUDED.INSERTED_AT_UTC
	`

	_, err := db.Exec(context.Background(), query, deviceID, startTime, endTime)
	if err != nil {
		return -1, fmt.Errorf("could not process CPU usage data for device %s: %w", deviceID, err)
	}

	err = UpdateProcessingPointer(deviceID, "cpu_usage", nextHour)
	if err != nil {
		return -1, fmt.Errorf("could not update processing pointer for device %s and parameter cpu_usage: %w", deviceID, err)
	}

	return 1, nil
}

func processDiskUsageData(deviceID string, startTime time.Time) (int, error) {
	nextHour := startTime.Add(time.Hour)
	endTime := nextHour.Add(-1 * time.Second)

	query := `
			INSERT INTO DISK_USAGE_HOURLY (
					ID_DEVICE,
					DISK_NAME,
					START_TIME,
					AVG_USED_PERCENT,
					MIN_USED_PERCENT,
					MAX_USED_PERCENT,
					TOTAL_DISK,
					ROW_COUNT,
					INSERTED_AT_UTC
			)
			SELECT 
					ID_DEVICE,
					DISK_NAME,
					$2 AS START_TIME,
					AVG(USED_PERCENT) AS AVG_USED_PERCENT,
					MIN(USED_PERCENT) AS MIN_USED_PERCENT,
					MAX(USED_PERCENT) AS MAX_USED_PERCENT,
					SUM(TOTAL_DISK) AS TOTAL_DISK,
					COUNT(*) AS ROW_COUNT,
					CURRENT_TIMESTAMP AS INSERTED_AT_UTC
			FROM DISK_USAGE
			WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC BETWEEN $2 AND $3
			GROUP BY ID_DEVICE, DISK_NAME
			ON CONFLICT (ID_DEVICE, DISK_NAME, START_TIME)
			DO UPDATE SET 
					AVG_USED_PERCENT = EXCLUDED.AVG_USED_PERCENT,
					MIN_USED_PERCENT = EXCLUDED.MIN_USED_PERCENT,
					MAX_USED_PERCENT = EXCLUDED.MAX_USED_PERCENT,
					TOTAL_DISK = EXCLUDED.TOTAL_DISK,
					ROW_COUNT = EXCLUDED.ROW_COUNT,
					INSERTED_AT_UTC = EXCLUDED.INSERTED_AT_UTC
	`

	_, err := db.Exec(context.Background(), query, deviceID, startTime, endTime)
	if err != nil {
		return -1, fmt.Errorf("could not process disk usage data for device %s: %w", deviceID, err)
	}

	err = UpdateProcessingPointer(deviceID, "disk", nextHour)
	if err != nil {
		return -1, fmt.Errorf("could not update processing pointer for device %s and parameter disk_usage: %w", deviceID, err)
	}

	return 1, nil
}

func processRAMUsageData(deviceID string, startTime time.Time) (int, error) {
	nextHour := startTime.Add(time.Hour)
	endTime := nextHour.Add(-1 * time.Second)

	query := `
			INSERT INTO RAM_USAGE_HOURLY (
					ID_DEVICE,
					START_TIME,
					AVG_USED_PERCENT_RAM,
					MIN_USED_PERCENT_RAM,
					MAX_USED_PERCENT_RAM,
					TOTAL_RAM,
					ROW_COUNT,
					INSERTED_AT_UTC
			)
			SELECT 
					ID_DEVICE,
					$2 AS START_TIME,
					AVG(USED_PERCENT_RAM) AS AVG_USED_PERCENT_RAM,
					MIN(USED_PERCENT_RAM) AS MIN_USED_PERCENT_RAM,
					MAX(USED_PERCENT_RAM) AS MAX_USED_PERCENT_RAM,
					SUM(TOTAL_RAM) AS TOTAL_RAM,
					COUNT(*) AS ROW_COUNT,
					CURRENT_TIMESTAMP AS INSERTED_AT_UTC
			FROM RAM_USAGE
			WHERE ID_DEVICE = $1 AND COLLECTED_AT_UTC BETWEEN $2 AND $3
			GROUP BY ID_DEVICE
			ON CONFLICT (ID_DEVICE, START_TIME)
			DO UPDATE SET 
					AVG_USED_PERCENT_RAM = EXCLUDED.AVG_USED_PERCENT_RAM,
					MIN_USED_PERCENT_RAM = EXCLUDED.MIN_USED_PERCENT_RAM,
					MAX_USED_PERCENT_RAM = EXCLUDED.MAX_USED_PERCENT_RAM,
					TOTAL_RAM = EXCLUDED.TOTAL_RAM,
					ROW_COUNT = EXCLUDED.ROW_COUNT,
					INSERTED_AT_UTC = EXCLUDED.INSERTED_AT_UTC
	`

	_, err := db.Exec(context.Background(), query, deviceID, startTime, endTime)
	if err != nil {
		return -1, fmt.Errorf("could not process RAM usage data for device %s: %w", deviceID, err)
	}

	err = UpdateProcessingPointer(deviceID, "ram", nextHour)
	if err != nil {
		return -1, fmt.Errorf("could not update processing pointer for device %s and parameter ram_usage: %w", deviceID, err)
	}

	return 1, nil
}

func UpdateProcessingPointer(deviceID, param string, nextHour time.Time) error {
	_, err := db.Exec(context.Background(), `
		INSERT INTO PROCESSING_POINTERS_HOURLY (ID_DEVICE, ID_PARAMETER, LAST_PROCESSED_AT)
		VALUES ($1, $2, $3)
		ON CONFLICT (ID_DEVICE, ID_PARAMETER)
		DO UPDATE SET LAST_PROCESSED_AT = $3
	`, deviceID, param, nextHour)
	return err
}
