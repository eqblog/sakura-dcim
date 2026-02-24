package service

import (
	"testing"
	"time"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

func TestCalculate95thPercentile(t *testing.T) {
	data := make([]domain.BandwidthDataPoint, 100)
	for i := 0; i < 100; i++ {
		data[i] = domain.BandwidthDataPoint{
			Timestamp: time.Now(),
			InBps:     float64(i + 1),
			OutBps:    float64((i + 1) * 2),
		}
	}

	in95 := calculate95thPercentile(data, true)
	if in95 != 95 {
		t.Errorf("expected 95th percentile = 95, got %v", in95)
	}

	out95 := calculate95thPercentile(data, false)
	if out95 != 190 {
		t.Errorf("expected 95th percentile = 190, got %v", out95)
	}
}

func TestCalculate95thPercentile_Empty(t *testing.T) {
	result := calculate95thPercentile(nil, true)
	if result != 0 {
		t.Errorf("expected 0 for empty data, got %v", result)
	}
}

func TestCalculate95thPercentile_Single(t *testing.T) {
	data := []domain.BandwidthDataPoint{
		{Timestamp: time.Now(), InBps: 42, OutBps: 84},
	}
	result := calculate95thPercentile(data, true)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestCalculateAvg(t *testing.T) {
	data := []domain.BandwidthDataPoint{
		{InBps: 10, OutBps: 20},
		{InBps: 20, OutBps: 40},
		{InBps: 30, OutBps: 60},
	}

	inAvg, outAvg := calculateAvg(data)
	if inAvg != 20 {
		t.Errorf("expected in avg = 20, got %v", inAvg)
	}
	if outAvg != 40 {
		t.Errorf("expected out avg = 40, got %v", outAvg)
	}
}

func TestCalculateAvg_Empty(t *testing.T) {
	inAvg, outAvg := calculateAvg(nil)
	if inAvg != 0 || outAvg != 0 {
		t.Errorf("expected 0,0 for empty, got %v,%v", inAvg, outAvg)
	}
}

func TestCalculateMax(t *testing.T) {
	data := []domain.BandwidthDataPoint{
		{InBps: 5, OutBps: 10},
		{InBps: 100, OutBps: 50},
		{InBps: 50, OutBps: 200},
	}

	inMax, outMax := calculateMax(data)
	if inMax != 100 {
		t.Errorf("expected in max = 100, got %v", inMax)
	}
	if outMax != 200 {
		t.Errorf("expected out max = 200, got %v", outMax)
	}
}

func TestFilterByPeriod(t *testing.T) {
	now := time.Now()
	data := []domain.BandwidthDataPoint{
		{Timestamp: now.Add(-48 * time.Hour), InBps: 1},  // 2 days ago
		{Timestamp: now.Add(-12 * time.Hour), InBps: 2},  // 12h ago
		{Timestamp: now.Add(-1 * time.Hour), InBps: 3},   // 1h ago
	}

	hourly := filterByPeriod(data, "hourly") // last 24h
	if len(hourly) != 2 {
		t.Errorf("expected 2 data points for hourly, got %d", len(hourly))
	}

	daily := filterByPeriod(data, "daily") // last 30 days
	if len(daily) != 3 {
		t.Errorf("expected 3 data points for daily, got %d", len(daily))
	}
}

func TestPeriodToTimeRange(t *testing.T) {
	tests := []struct {
		period       string
		expectedDays int
	}{
		{"hourly", 1},
		{"daily", 30},
		{"monthly", 365},
		{"unknown", 1},
	}

	for _, tt := range tests {
		start, stop := periodToTimeRange(tt.period)
		duration := stop.Sub(start)
		expectedDuration := time.Duration(tt.expectedDays) * 24 * time.Hour
		diff := duration - expectedDuration
		if diff < -time.Second || diff > time.Second {
			t.Errorf("period %q: expected ~%d days, got %v", tt.period, tt.expectedDays, duration)
		}
	}
}
