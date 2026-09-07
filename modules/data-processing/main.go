package main

import (
	"ceiot-tf-background/modules/data-processing/config"
	"ceiot-tf-background/modules/data-processing/models"
	"ceiot-tf-background/modules/data-processing/postgres"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const refreshInterval = 15 * time.Minute

const firstRefreshRetryInterval = 30 * time.Second

var (
	cfg      *models.Config
	routines map[string]chan struct{}
	mu       sync.Mutex
	wg       sync.WaitGroup
)

func main() {
	loadConfiguration()
	initializeDatabase()
	initializeProcessing()
	waitForShutdown()
}

func loadConfiguration() {
	var err error
	cfg, err = config.LoadEnvVars()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}
}

func initializeDatabase() {
	if err := postgres.ConnectDB(cfg.PostgresURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
}

func initializeProcessing() {
	routines = make(map[string]chan struct{})

	go func() {
		waitForFirstSuccessfulRefresh()

		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := refreshProcessingRoutines(); err != nil {
				log.Printf("Error actualizando rutinas de procesamiento, se reintentará en %s: %v", refreshInterval, err)
			}
		}
	}()
}

func waitForFirstSuccessfulRefresh() {
	for {
		if err := refreshProcessingRoutines(); err == nil {
			return
		}
		log.Printf("Reintentando la primera actualización de rutinas de procesamiento en %s...", firstRefreshRetryInterval)
		time.Sleep(firstRefreshRetryInterval)
	}
}

func refreshProcessingRoutines() error {
	devices, err := postgres.GetActiveDevices()
	if err != nil {
		log.Printf("Error fetching devices: %v", err)
		return err
	}

	parameters, err := postgres.GetActiveParameters()
	if err != nil {
		log.Printf("Error fetching parameters: %v", err)
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	activeRoutines := make(map[string]bool)

	for _, deviceID := range devices {
		for _, param := range parameters {
			key := deviceID + "_" + param
			activeRoutines[key] = true
			if _, exists := routines[key]; !exists {
				stopChan := make(chan struct{})
				routines[key] = stopChan
				wg.Add(1)
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

	return nil
}

func startProcessingForDeviceParameter(deviceID, param string, stopChan chan struct{}) {
	defer wg.Done()
	for {
		select {
		case <-stopChan:
			log.Printf("Stopping data processing for device %s and parameter %s", deviceID, param)
			return
		default:
			status, err := postgres.ProcessParameterData(deviceID, param)
			if err != nil {
				log.Printf("Error processing data for device %s and parameter %s: %v", deviceID, param, err)
				log.Printf("Retrying in 1 hour for device %s and parameter %s", deviceID, param)
				if !sleepOrStop(1*time.Hour, stopChan) {
					return
				}
				continue
			}
			if status == 0 {
				log.Printf("No data to process for device %s and parameter %s. Retrying in 1 hour.", deviceID, param)
				if !sleepOrStop(1*time.Hour, stopChan) {
					return
				}
			} else if status == 1 {
				log.Printf("Successfully processed data for device %s and parameter %s", deviceID, param)
			}
		}
	}
}

func sleepOrStop(d time.Duration, stopChan chan struct{}) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-stopChan:
		return false
	}
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Señal de apagado recibida, deteniendo rutinas de procesamiento...")
	mu.Lock()
	for key, stopChan := range routines {
		close(stopChan)
		delete(routines, key)
	}
	mu.Unlock()

	wg.Wait()
	postgres.CloseDB()
	log.Println("Apagado completo.")
}
