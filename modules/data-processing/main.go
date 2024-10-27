package main

import (
	"ceiot-tf-background/modules/data-processing/config"
	"ceiot-tf-background/modules/data-processing/models"
	"ceiot-tf-background/modules/data-processing/postgres"
	"log"
	"sync"
	"time"
)

var (
	err      error
	cfg      *models.Config
	routines map[string]chan bool
	mu       sync.Mutex
)

func main() {
	loadConfiguration()
	initializeDatabase()
	initializeProcessing()
	defer postgres.CloseDB()
	select {}
}

func loadConfiguration() {
	cfg, err = config.LoadEnvVars()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}
}

func initializeDatabase() {
	err = postgres.ConnectDB(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
}

func initializeProcessing() {
	routines = make(map[string]chan bool)
	go func() {
		for {
			refreshProcessingRoutines()
			time.Sleep(15 * time.Minute)
		}
	}()
}

func refreshProcessingRoutines() {
	devices, err := postgres.GetActiveDevices()
	if err != nil {
		log.Fatalf("Error fetching devices: %v", err)
	}

	parameters, err := postgres.GetActiveParameters()
	if err != nil {
		log.Fatalf("Error fetching parameters: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	activeRoutines := make(map[string]bool)

	for _, deviceID := range devices {
		for _, param := range parameters {
			key := deviceID + "_" + param
			activeRoutines[key] = true
			if _, exists := routines[key]; !exists {
				stopChan := make(chan bool)
				routines[key] = stopChan
				go startProcessingForDeviceParameter(deviceID, param, stopChan)
			}
		}
	}

	for key, stopChan := range routines {
		if !activeRoutines[key] {
			close(stopChan)
			delete(routines, key)
		}
	}
}

func startProcessingForDeviceParameter(deviceID, param string, stopChan chan bool) {
	for {
		select {
		case <-stopChan:
			log.Printf("Stopping data processing for device %s and parameter %s", deviceID, param)
			return
		default:
			status, err := postgres.ProcessParameterData(deviceID, param)
			if err != nil {
				log.Printf("Error processing data for device %s and parameter %s: %v", deviceID, param, err)
				time.Sleep(time.Hour)
			}
			if status == 0 {
				time.Sleep(time.Hour)
			}
			if status == 1 {
				log.Printf("Successfully processed data for device %s and parameter %s", deviceID, param)
			}
		}
	}
}
