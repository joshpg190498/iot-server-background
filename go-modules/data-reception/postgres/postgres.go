package postgres

import (
	"ceiot-tf-background/go-modules/data-reception/models"
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"

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

func insertRAMUsage(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
		INSERT INTO RAM_USAGE (
			ID_DEVICE, TOTAL_RAM, FREE_RAM, USED_RAM, USED_PERCENT_RAM, COLLECTED_AT_UTC
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	_, err := tx.Exec(
		context.Background(),
		query,
		dataPayload.IDDevice,
		dataMap["totalRAM"],
		dataMap["freeRAM"],
		dataMap["usedRAM"],
		dataMap["usedPercentRAM"],
		dataPayload.CollectedAtUtc)
	if err != nil {
		return err
	}

	log.Printf("RAM usage data inserted for device %s\n", dataPayload.IDDevice)
	return nil
}

func insertDiskUsage(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
		INSERT INTO DISK_USAGE (
			ID_DEVICE, DISK_NAME, TOTAL_DISK, FREE_DISK, USED_DISK, USED_PERCENT_DISK, COLLECTED_AT_UTC
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	for diskName, diskData := range dataMap {
		data := diskData.(map[string]interface{})
		_, err := tx.Exec(
			context.Background(),
			query,
			dataPayload.IDDevice,
			diskName,
			data["totalDisk"],
			data["freeDisk"],
			data["usedDisk"],
			data["usedPercentDisk"],
			dataPayload.CollectedAtUtc)
		if err != nil {
			return fmt.Errorf("failed to insert disk usage: %w", err)
		}
		log.Printf("Disk usage data inserted for device %s, disk %s\n", dataPayload.IDDevice, diskName)
	}
	return nil
}

func insertNetworkStats(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
		INSERT INTO NETWORK_STATS (
			ID_DEVICE, INTERFACE_NAME, BYTES_SENT, BYTES_RECV, PACKETS_SENT, PACKETS_RECV, ERROUT, ERRIN, DROPIN, DROPOUT, COLLECTED_AT_UTC
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	for ifaceName, ifaceData := range dataMap {
		data := ifaceData.(map[string]interface{})
		_, err := tx.Exec(
			context.Background(),
			query,
			dataPayload.IDDevice,
			ifaceName,
			data["bytesSent"],
			data["bytesRecv"],
			data["packetsSent"],
			data["packetsRecv"],
			data["errout"],
			data["errin"],
			data["dropin"],
			data["dropout"],
			dataPayload.CollectedAtUtc)
		if err != nil {
			return fmt.Errorf("failed to insert network stats: %w", err)
		}
		log.Printf("Network stats data inserted for device %s, ifaceName %s\n", dataPayload.IDDevice, ifaceName)
	}
	return nil
}

func insertNetworkInfo(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
	INSERT INTO NETWORK_INFORMATION (
		ID_DEVICE, INTERFACE_NAME, MTU, HARDWARE_ADDR, FLAGS, ADDRS, COLLECTED_AT_UTC
	) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	for ifaceName, ifaceData := range dataMap {
		data := ifaceData.(map[string]interface{})
		flags := fmt.Sprintf("%v", data["flags"])
		addrs := fmt.Sprintf("%v", data["addrs"])
		_, err := tx.Exec(
			context.Background(),
			query,
			dataPayload.IDDevice,
			ifaceName,
			data["mtu"],
			data["hardwareAddr"],
			flags,
			addrs,
			dataPayload.CollectedAtUtc)
		if err != nil {
			return fmt.Errorf("failed to insert network info: %w", err)
		}
		log.Printf("Network information data inserted for device %s, interface %s\n", dataPayload.IDDevice, ifaceName)
	}
	return nil
}

func insertCPUTemperature(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
		INSERT INTO CPU_TEMPERATURE (
			ID_DEVICE, SENSOR_KEY, TEMPERATURE, COLLECTED_AT_UTC
		) VALUES ($1, $2, $3, $4)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	for sensorKey, sensorData := range dataMap {
		_, err := tx.Exec(
			context.Background(),
			query,
			dataPayload.IDDevice,
			sensorKey,
			sensorData,
			dataPayload.CollectedAtUtc)
		if err != nil {
			return fmt.Errorf("failed to insert CPU temperature: %w", err)
		}
		log.Printf("CPU temperature data inserted for device %s, sensor %s\n", dataPayload.IDDevice, sensorKey)
	}

	return nil
}

func insertUptime(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
		INSERT INTO UPTIME (
			ID_DEVICE, UPTIME_MINUTES, COLLECTED_AT_UTC
		) VALUES ($1, $2, $3)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	_, err := tx.Exec(context.Background(), query, dataPayload.IDDevice, dataMap["uptime"], dataPayload.CollectedAtUtc)
	if err != nil {
		return fmt.Errorf("failed to insert uptime: %w", err)
	}

	log.Printf("Uptime data inserted for device %s\n", dataPayload.IDDevice)
	return nil
}

func insertLastReboot(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
		INSERT INTO LAST_REBOOT (
			ID_DEVICE, LAST_REBOOT, COLLECTED_AT_UTC
		) VALUES ($1, $2, $3)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	_, err := tx.Exec(context.Background(), query, dataPayload.IDDevice, dataMap["lastReboot"], dataPayload.CollectedAtUtc)
	if err != nil {
		return fmt.Errorf("failed to insert last reboot: %w", err)
	}

	log.Printf("Last reboot data inserted for device %s\n", dataPayload.IDDevice)
	return nil
}

func insertCPUUsage(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
		INSERT INTO CPU_USAGE (
			ID_DEVICE, CPU_USAGE, COLLECTED_AT_UTC
		) VALUES ($1, $2, $3)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	_, err := tx.Exec(context.Background(), query, dataPayload.IDDevice, dataMap["cpuUsage"], dataPayload.CollectedAtUtc)
	if err != nil {
		return fmt.Errorf("failed to insert CPU usage: %w", err)
	}

	log.Printf("CPU usage data inserted for device %s\n", dataPayload.IDDevice)
	return nil
}

func insertLoadAverage(tx pgx.Tx, dataPayload models.DataPayload) error {
	query := `
		INSERT INTO LOAD_AVERAGE (
			ID_DEVICE, LOAD_AVERAGE_1M, LOAD_AVERAGE_5M, LOAD_AVERAGE_15M, COLLECTED_AT_UTC
		) VALUES ($1, $2, $3, $4, $5)
	`

	dataMap, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data payload format")
	}

	_, err := tx.Exec(
		context.Background(),
		query,
		dataPayload.IDDevice,
		dataMap["loadAverage1m"],
		dataMap["loadAverage5m"],
		dataMap["loadAverage15m"],
		dataPayload.CollectedAtUtc)
	if err != nil {
		return fmt.Errorf("failed to insert load average: %w", err)
	}

	log.Printf("Load average data inserted for device %s\n", dataPayload.IDDevice)
	return nil
}

type FuncType func(tx pgx.Tx, dataPayload models.DataPayload) error

func getInsertDataFunctions() map[string]FuncType {
	return map[string]FuncType{
		"ram":          insertRAMUsage,
		"disk":         insertDiskUsage,
		"net_stats":    insertNetworkStats,
		"net_info":     insertNetworkInfo,
		"cpu_temp":     insertCPUTemperature,
		"uptime":       insertUptime,
		"last_reboot":  insertLastReboot,
		"cpu_usage":    insertCPUUsage,
		"load_average": insertLoadAverage,
	}
}

func InsertData(dataPayload models.DataPayload) error {
	ctx := context.Background()
	if dataPayload.Data == nil {
		return errors.New("empty data payload")
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	functions := getInsertDataFunctions()
	if function, exists := functions[dataPayload.Parameter]; exists {
		err := function(tx, dataPayload)
		if err != nil {
			return err
		}
	} else {
		return errors.New("function not found")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	log.Println("Data insertion successful")
	return nil
}

func hasNoData(data interface{}) bool {
	v := reflect.ValueOf(data)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.IsZero() {
			return false
		}
	}
	return true
}
