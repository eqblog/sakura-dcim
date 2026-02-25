package service

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	influxRepo "github.com/sakura-dcim/sakura-dcim/backend/internal/repository/influxdb"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

type BandwidthService struct {
	switchRepo repository.SwitchRepository
	portRepo   repository.SwitchPortRepository
	hub        *ws.Hub
	logger     *zap.Logger
	influxRepo *influxRepo.BandwidthRepo
	// In-memory fallback when InfluxDB is unavailable
	dataStore map[string][]domain.BandwidthDataPoint
}

func NewBandwidthService(switchRepo repository.SwitchRepository, portRepo repository.SwitchPortRepository, hub *ws.Hub, logger *zap.Logger) *BandwidthService {
	return &BandwidthService{
		switchRepo: switchRepo,
		portRepo:   portRepo,
		hub:        hub,
		logger:     logger,
		dataStore:  make(map[string][]domain.BandwidthDataPoint),
	}
}

func (s *BandwidthService) SetInfluxRepo(repo *influxRepo.BandwidthRepo) {
	s.influxRepo = repo
}

// HandleSNMPDataEvent processes SNMP data events from agents.
func (s *BandwidthService) HandleSNMPDataEvent(agentID uuid.UUID, msg *ws.Message) {
	s.logger.Debug("received SNMP data", zap.String("agent_id", agentID.String()))

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return
	}

	var data struct {
		SwitchID string `json:"switch_id"`
		Ports    []struct {
			PortID   string  `json:"port_id"`
			InBytes  uint64  `json:"in_bytes"`
			OutBytes uint64  `json:"out_bytes"`
			InBps    float64 `json:"in_bps"`
			OutBps   float64 `json:"out_bps"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(payloadBytes, &data); err != nil {
		s.logger.Error("failed to parse SNMP data", zap.Error(err))
		return
	}

	ctx := context.Background()
	for _, port := range data.Ports {
		dp := domain.BandwidthDataPoint{
			Timestamp: time.Now(),
			InBytes:   port.InBytes,
			OutBytes:  port.OutBytes,
			InBps:     port.InBps,
			OutBps:    port.OutBps,
		}

		// Write to InfluxDB if available
		if s.influxRepo != nil {
			if err := s.influxRepo.WriteBandwidthPoint(ctx, port.PortID, data.SwitchID, port.InBytes, port.OutBytes, port.InBps, port.OutBps); err != nil {
				s.logger.Error("failed to write bandwidth to InfluxDB", zap.Error(err))
			}
		}

		// Also store in memory as fallback
		s.dataStore[port.PortID] = append(s.dataStore[port.PortID], dp)
		if len(s.dataStore[port.PortID]) > 8640 {
			s.dataStore[port.PortID] = s.dataStore[port.PortID][len(s.dataStore[port.PortID])-8640:]
		}
	}
}

// GetServerBandwidth returns bandwidth data for a server's switch ports.
func (s *BandwidthService) GetServerBandwidth(ctx context.Context, serverID uuid.UUID, period string) ([]domain.BandwidthSummary, error) {
	ports, err := s.portRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	var summaries []domain.BandwidthSummary
	for _, port := range ports {
		summary := domain.BandwidthSummary{
			PortID:    port.ID,
			PortName:  port.PortName,
			ServerID:  port.ServerID,
			SpeedMbps: port.SpeedMbps,
		}

		var data []domain.BandwidthDataPoint

		// Try InfluxDB first
		if s.influxRepo != nil {
			start, stop := periodToTimeRange(period)
			if influxData, err := s.influxRepo.QueryBandwidth(ctx, port.ID.String(), start, stop); err == nil && len(influxData) > 0 {
				data = influxData
			}
		}

		// Fall back to in-memory
		if len(data) == 0 {
			key := port.ID.String()
			if memData, ok := s.dataStore[key]; ok && len(memData) > 0 {
				data = filterByPeriod(memData, period)
			}
		}

		if len(data) > 0 {
			summary.DataPoints = data
			summary.In95th = calculate95thPercentile(data, true)
			summary.Out95th = calculate95thPercentile(data, false)
			summary.InAvg, summary.OutAvg = calculateAvg(data)
			summary.InMax, summary.OutMax = calculateMax(data)
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// GetSwitchBandwidth returns traffic today/month for all ports of a switch.
func (s *BandwidthService) GetSwitchBandwidth(ctx context.Context, switchID uuid.UUID) (domain.SwitchBandwidthMap, error) {
	ports, err := s.portRepo.ListBySwitchID(ctx, switchID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	result := make(domain.SwitchBandwidthMap)
	for _, port := range ports {
		summary := domain.PortTrafficSummary{}
		portID := port.ID.String()

		if s.influxRepo != nil {
			if todayData, err := s.influxRepo.QueryBandwidth(ctx, portID, todayStart, now); err == nil {
				for _, dp := range todayData {
					summary.TrafficTodayIn += dp.InBytes
					summary.TrafficTodayOut += dp.OutBytes
				}
			}
			if monthData, err := s.influxRepo.QueryBandwidth(ctx, portID, monthStart, now); err == nil {
				for _, dp := range monthData {
					summary.TrafficMonthIn += dp.InBytes
					summary.TrafficMonthOut += dp.OutBytes
				}
			}
		} else if memData, ok := s.dataStore[portID]; ok {
			for _, dp := range memData {
				if dp.Timestamp.After(todayStart) {
					summary.TrafficTodayIn += dp.InBytes
					summary.TrafficTodayOut += dp.OutBytes
				}
				if dp.Timestamp.After(monthStart) {
					summary.TrafficMonthIn += dp.InBytes
					summary.TrafficMonthOut += dp.OutBytes
				}
			}
		}

		result[portID] = summary
	}
	return result, nil
}

// TriggerSNMPPoll sends poll request to the agent for a specific switch.
func (s *BandwidthService) TriggerSNMPPoll(ctx context.Context, switchID uuid.UUID) error {
	sw, err := s.switchRepo.GetByID(ctx, switchID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"switch_ip":      sw.IP,
		"snmp_community": sw.SNMPCommunity,
		"snmp_version":   sw.SNMPVersion,
	}

	_, err = s.hub.SendRequest(sw.AgentID, ws.ActionSNMPPoll, payload, 15*time.Second)
	return err
}

func periodToTimeRange(period string) (time.Time, time.Time) {
	now := time.Now()
	switch period {
	case "hourly":
		return now.Add(-24 * time.Hour), now
	case "daily":
		return now.Add(-30 * 24 * time.Hour), now
	case "monthly":
		return now.Add(-365 * 24 * time.Hour), now
	default:
		return now.Add(-24 * time.Hour), now
	}
}

func filterByPeriod(data []domain.BandwidthDataPoint, period string) []domain.BandwidthDataPoint {
	start, _ := periodToTimeRange(period)
	var filtered []domain.BandwidthDataPoint
	for _, dp := range data {
		if dp.Timestamp.After(start) {
			filtered = append(filtered, dp)
		}
	}
	return filtered
}

func calculate95thPercentile(data []domain.BandwidthDataPoint, isIn bool) float64 {
	if len(data) == 0 {
		return 0
	}
	values := make([]float64, len(data))
	for i, dp := range data {
		if isIn {
			values[i] = dp.InBps
		} else {
			values[i] = dp.OutBps
		}
	}
	sort.Float64s(values)
	idx := int(math.Ceil(float64(len(values))*0.95)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}

func calculateAvg(data []domain.BandwidthDataPoint) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}
	var inSum, outSum float64
	for _, dp := range data {
		inSum += dp.InBps
		outSum += dp.OutBps
	}
	n := float64(len(data))
	return inSum / n, outSum / n
}

func calculateMax(data []domain.BandwidthDataPoint) (float64, float64) {
	var inMax, outMax float64
	for _, dp := range data {
		if dp.InBps > inMax {
			inMax = dp.InBps
		}
		if dp.OutBps > outMax {
			outMax = dp.OutBps
		}
	}
	return inMax, outMax
}
