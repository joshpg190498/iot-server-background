package evaluator

import (
	"ceiot-tf-background/modules/threshold-validator/models"
	"errors"
	"fmt"
)

func GetThresholdExceededData(setting models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	if setting.ThresholdValue == nil {
		return nil, fmt.Errorf("threshold value no configurado para device=%s parameter=%s pese a HasThreshold=true", setting.IDDevice, setting.Parameter)
	}

	functions := getEvaluatorsFunction()
	function, exists := functions[dataPayload.Parameter]
	if !exists {
		return nil, errors.New("function not found")
	}
	return function(setting, dataPayload)
}

func evaluateCPUTemperature(setting models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data payload format for cpu_temp")
	}
	threshold := *setting.ThresholdValue
	exceeded := []models.ThresholdExceededData{}

	for sensorKey, temperature := range data {
		if sensorKey == "cpu_thermal_crit" {
			continue
		}
		tempValue, ok := temperature.(float64)
		if !ok {
			continue // valor no numérico para este sensor puntual: se ignora
		}
		if tempValue > threshold {
			exceeded = append(exceeded, models.ThresholdExceededData{Key: sensorKey, Value: tempValue})
		}
	}
	return exceeded, nil
}

func evaluateDiskUsage(setting models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data payload format for disk")
	}
	threshold := *setting.ThresholdValue
	exceeded := []models.ThresholdExceededData{}

	for diskName, diskData := range data {
		diskMap, ok := diskData.(map[string]interface{})
		if !ok {
			continue
		}
		usedPercentDisk, ok := diskMap["usedPercentDisk"].(float64)
		if !ok {
			continue
		}
		if usedPercentDisk > threshold {
			exceeded = append(exceeded, models.ThresholdExceededData{Key: diskName, Value: usedPercentDisk})
		}
	}
	return exceeded, nil
}

func evaluateCPUUsage(setting models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data payload format for cpu_usage")
	}
	cpuUsage, ok := data["cpuUsage"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid cpuUsage value")
	}

	exceeded := []models.ThresholdExceededData{}
	if cpuUsage > *setting.ThresholdValue {
		exceeded = append(exceeded, models.ThresholdExceededData{Key: "cpuUsage", Value: cpuUsage})
	}
	return exceeded, nil
}

func evaluateRAMUsage(setting models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data payload format for ram")
	}
	usedPercentRAM, ok := data["usedPercentRAM"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid usedPercentRAM value")
	}

	exceeded := []models.ThresholdExceededData{}
	if usedPercentRAM > *setting.ThresholdValue {
		exceeded = append(exceeded, models.ThresholdExceededData{Key: "usedPercentRAM", Value: usedPercentRAM})
	}
	return exceeded, nil
}

func evaluateLoadAverage(setting models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data, ok := dataPayload.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data payload format for load_average")
	}
	loadAverage5m, ok := data["loadAverage5m"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid loadAverage5m value")
	}

	exceeded := []models.ThresholdExceededData{}
	if loadAverage5m > *setting.ThresholdValue {
		exceeded = append(exceeded, models.ThresholdExceededData{Key: "loadAverage5m", Value: loadAverage5m})
	}
	return exceeded, nil
}

type FuncType func(setting models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error)

func getEvaluatorsFunction() map[string]FuncType {
	return map[string]FuncType{
		"ram":          evaluateRAMUsage,
		"disk":         evaluateDiskUsage,
		"cpu_temp":     evaluateCPUTemperature,
		"cpu_usage":    evaluateCPUUsage,
		"load_average": evaluateLoadAverage,
	}
}
