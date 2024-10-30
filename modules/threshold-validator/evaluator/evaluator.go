package evaluator

import (
	"ceiot-tf-background/modules/threshold-validator/models"
	"errors"
)

func GetThresholdExceededData(setting models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	functions := getEvaluatorsFunction()
	if function, exists := functions[dataPayload.Parameter]; exists {
		thresholdExceededData, err := function(setting, dataPayload)
		if err != nil {
			return nil, err
		}
		return thresholdExceededData, nil
	} else {
		return nil, errors.New("function not found")
	}
}

func evaluateCPUTemperature(evaluator models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data := dataPayload.Data.(map[string]interface{})
	exceeded := []models.ThresholdExceededData{}

	for sensorKey, temperature := range data {
		if sensorKey == "cpu_thermal_crit" {
			continue
		}
		tempValue := temperature.(float64)
		if tempValue > float64(*evaluator.ThresholdValue) {
			exceeded = append(exceeded, models.ThresholdExceededData{Key: sensorKey, Value: tempValue})
		}
	}
	return exceeded, nil
}

func evaluateDiskUsage(evaluator models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data := dataPayload.Data.(map[string]interface{})
	exceeded := []models.ThresholdExceededData{}

	for diskName, diskData := range data {
		usedPercentDisk := diskData.(map[string]interface{})["usedPercentDisk"].(float64)
		if usedPercentDisk > float64(*evaluator.ThresholdValue) {
			exceeded = append(exceeded, models.ThresholdExceededData{Key: diskName, Value: usedPercentDisk})
		}
	}
	return exceeded, nil
}

func evaluateCPUUsage(evaluator models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data := dataPayload.Data.(map[string]interface{})
	cpuUsage := data["cpuUsage"].(float64)
	exceeded := []models.ThresholdExceededData{}

	if cpuUsage > float64(*evaluator.ThresholdValue) {
		exceeded = append(exceeded, models.ThresholdExceededData{Key: "cpuUsage", Value: cpuUsage})
	}

	return exceeded, nil
}

func evaluateRAMUsage(evaluator models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data := dataPayload.Data.(map[string]interface{})
	usedPercentRAM := data["usedPercentRAM"].(float64)
	exceeded := []models.ThresholdExceededData{}

	if usedPercentRAM > float64(*evaluator.ThresholdValue) {
		exceeded = append(exceeded, models.ThresholdExceededData{Key: "usedPercentRAM", Value: usedPercentRAM})
	}

	return exceeded, nil
}
func evaluateLoadAverage(evaluator models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error) {
	data := dataPayload.Data.(map[string]interface{})
	loadAverage5m := data["loadAverage5m"].(float64)
	exceeded := []models.ThresholdExceededData{}

	if loadAverage5m > float64(*evaluator.ThresholdValue) {
		exceeded = append(exceeded, models.ThresholdExceededData{Key: "loadAverage5m", Value: loadAverage5m})
	}

	return exceeded, nil
}

type FuncType func(evaluator models.DeviceReadingSetting, dataPayload models.DataPayload) ([]models.ThresholdExceededData, error)

func getEvaluatorsFunction() map[string]FuncType {
	return map[string]FuncType{
		"ram":          evaluateRAMUsage,
		"disk":         evaluateDiskUsage,
		"cpu_temp":     evaluateCPUTemperature,
		"cpu_usage":    evaluateCPUUsage,
		"load_average": evaluateLoadAverage,
	}
}
