package service

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

type BandwidthService struct {
	switchRepo repository.SwitchRepository
	portRepo   repository.SwitchPortRepository
	hub        *ws.Hub
	logger     *zap.Logger
	// In-memory bandwidth data store (replace with InfluxDB when available)
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

// HandleSNMPDataEvent processes SNMP data events from agents.
func (s *BandwidthService) HandleSNMPDataEvent(agentID uuid.UUID, msg *ws.Message) {
	// Parse SNMP data and store
	s.logger.Debug("received SNMP data", zap.String("agent_id", agentID.String()))
	// TODO: write to InfluxDB when client is available
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

		key := port.ID.String()
		if data, ok := s.dataStore[key]; ok && len(data) > 0 {
			summary.DataPoints = filterByPeriod(data, period)
			summary.In95th = calculate95thPercentile(summary.DataPoints, true)
			summary.Out95th = calculate95thPercentile(summary.DataPoints, false)
			summary.InAvg, summary.OutAvg = calculateAvg(summary.DataPoints)
			summary.InMax, summary.OutMax = calculateMax(summary.DataPoints)
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
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

func filterByPeriod(data []domain.BandwidthDataPoint, period string) []domain.BandwidthDataPoint {
	var cutoff time.Time
	now := time.Now()
	switch period {
	case "hourly":
		cutoff = now.Add(-24 * time.Hour)
	case "daily":
		cutoff = now.Add(-30 * 24 * time.Hour)
	case "monthly":
		cutoff = now.Add(-365 * 24 * time.Hour)
	default:
		cutoff = now.Add(-24 * time.Hour)
	}

	var filtered []domain.BandwidthDataPoint
	for _, dp := range data {
		if dp.Timestamp.After(cutoff) {
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
