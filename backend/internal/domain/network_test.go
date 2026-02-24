package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBandwidthDataPoint_JSON(t *testing.T) {
	dp := BandwidthDataPoint{
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		InBytes:   1000,
		OutBytes:  2000,
		InBps:     100.5,
		OutBps:    200.5,
	}

	data, err := json.Marshal(dp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded BandwidthDataPoint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.InBytes != 1000 {
		t.Errorf("expected InBytes=1000, got %d", decoded.InBytes)
	}
	if decoded.OutBps != 200.5 {
		t.Errorf("expected OutBps=200.5, got %v", decoded.OutBps)
	}
}

func TestSensorDataPoint_JSON(t *testing.T) {
	dp := SensorDataPoint{
		Timestamp:  time.Now(),
		SensorName: "CPU Temp",
		SensorType: "Temperature",
		Value:      65.5,
		Status:     "ok",
	}

	data, err := json.Marshal(dp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded SensorDataPoint
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.SensorName != "CPU Temp" {
		t.Errorf("expected sensor_name='CPU Temp', got %s", decoded.SensorName)
	}
	if decoded.Value != 65.5 {
		t.Errorf("expected value=65.5, got %v", decoded.Value)
	}
}

func TestBandwidthSummary_JSON(t *testing.T) {
	summary := BandwidthSummary{
		PortName:  "eth0",
		SpeedMbps: 10000,
		In95th:    500000,
		Out95th:   250000,
		InAvg:     300000,
		OutAvg:    150000,
		InMax:     800000,
		OutMax:    400000,
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}

	var decoded BandwidthSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.SpeedMbps != 10000 {
		t.Errorf("expected SpeedMbps=10000, got %d", decoded.SpeedMbps)
	}
	if decoded.In95th != 500000 {
		t.Errorf("expected In95th=500000, got %v", decoded.In95th)
	}
}
