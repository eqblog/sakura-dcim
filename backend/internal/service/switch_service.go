package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

type SwitchService struct {
	switchRepo repository.SwitchRepository
	portRepo   repository.SwitchPortRepository
	hub        *ws.Hub
	logger     *zap.Logger
}

func NewSwitchService(switchRepo repository.SwitchRepository, portRepo repository.SwitchPortRepository, hub *ws.Hub, logger *zap.Logger) *SwitchService {
	return &SwitchService{switchRepo: switchRepo, portRepo: portRepo, hub: hub, logger: logger}
}

// Switch CRUD

func (s *SwitchService) List(ctx context.Context) ([]domain.Switch, error) {
	return s.switchRepo.List(ctx)
}

func (s *SwitchService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Switch, error) {
	return s.switchRepo.GetByID(ctx, id)
}

func (s *SwitchService) Create(ctx context.Context, sw *domain.Switch) (*domain.Switch, error) {
	if sw.SSHPort == 0 {
		sw.SSHPort = 22
	}
	if sw.SNMPCommunity == "" {
		sw.SNMPCommunity = "public"
	}
	if sw.SNMPVersion == "" {
		sw.SNMPVersion = "v2c"
	}
	if err := s.switchRepo.Create(ctx, sw); err != nil {
		return nil, err
	}
	return sw, nil
}

func (s *SwitchService) Update(ctx context.Context, id uuid.UUID, sw *domain.Switch) (*domain.Switch, error) {
	sw.ID = id
	if err := s.switchRepo.Update(ctx, sw); err != nil {
		return nil, err
	}
	return s.switchRepo.GetByID(ctx, id)
}

func (s *SwitchService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.switchRepo.Delete(ctx, id)
}

// Port CRUD

func (s *SwitchService) ListPorts(ctx context.Context, switchID uuid.UUID) ([]domain.SwitchPort, error) {
	return s.portRepo.ListBySwitchID(ctx, switchID)
}

func (s *SwitchService) CreatePort(ctx context.Context, port *domain.SwitchPort) (*domain.SwitchPort, error) {
	if port.AdminStatus == "" {
		port.AdminStatus = "up"
	}
	if err := s.portRepo.Create(ctx, port); err != nil {
		return nil, err
	}
	return port, nil
}

func (s *SwitchService) UpdatePort(ctx context.Context, id uuid.UUID, port *domain.SwitchPort) (*domain.SwitchPort, error) {
	port.ID = id
	if err := s.portRepo.Update(ctx, port); err != nil {
		return nil, err
	}
	return s.portRepo.GetByID(ctx, id)
}

func (s *SwitchService) DeletePort(ctx context.Context, id uuid.UUID) error {
	return s.portRepo.Delete(ctx, id)
}

func (s *SwitchService) GetPortsByServerID(ctx context.Context, serverID uuid.UUID) ([]domain.SwitchPort, error) {
	return s.portRepo.GetByServerID(ctx, serverID)
}

// GetPortsWithSwitchInfo returns ports linked to a server, enriched with switch name/IP.
func (s *SwitchService) GetPortsWithSwitchInfo(ctx context.Context, serverID uuid.UUID) ([]domain.SwitchPortWithSwitch, error) {
	ports, err := s.portRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if len(ports) == 0 {
		return []domain.SwitchPortWithSwitch{}, nil
	}

	// Collect unique switch IDs
	switchIDs := make(map[uuid.UUID]struct{})
	for _, p := range ports {
		switchIDs[p.SwitchID] = struct{}{}
	}

	// Batch-fetch switch details
	switchMap := make(map[uuid.UUID]*domain.Switch)
	for id := range switchIDs {
		sw, err := s.switchRepo.GetByID(ctx, id)
		if err != nil {
			s.logger.Warn("failed to fetch switch for port enrichment", zap.String("switch_id", id.String()))
			continue
		}
		switchMap[id] = sw
	}

	// Build enriched result
	result := make([]domain.SwitchPortWithSwitch, 0, len(ports))
	for _, p := range ports {
		item := domain.SwitchPortWithSwitch{SwitchPort: p}
		if sw, ok := switchMap[p.SwitchID]; ok {
			item.SwitchName = sw.Name
			item.SwitchIP = sw.IP
		}
		result = append(result, item)
	}
	return result, nil
}

// LinkPortToServer sets the server_id on a switch port.
func (s *SwitchService) LinkPortToServer(ctx context.Context, portID, serverID uuid.UUID) error {
	port, err := s.portRepo.GetByID(ctx, portID)
	if err != nil {
		return fmt.Errorf("port not found: %w", err)
	}
	port.ServerID = &serverID
	return s.portRepo.Update(ctx, port)
}

// UnlinkPort clears the server_id from a switch port.
func (s *SwitchService) UnlinkPort(ctx context.Context, portID uuid.UUID) error {
	port, err := s.portRepo.GetByID(ctx, portID)
	if err != nil {
		return fmt.Errorf("port not found: %w", err)
	}
	port.ServerID = nil
	return s.portRepo.Update(ctx, port)
}

// ProvisionPort sends an SSH command to configure a switch port via the agent.
func (s *SwitchService) ProvisionPort(ctx context.Context, switchID uuid.UUID, portID uuid.UUID) error {
	sw, err := s.switchRepo.GetByID(ctx, switchID)
	if err != nil {
		return fmt.Errorf("switch not found: %w", err)
	}
	port, err := s.portRepo.GetByID(ctx, portID)
	if err != nil {
		return fmt.Errorf("port not found: %w", err)
	}

	payload := map[string]any{
		"switch_ip":   sw.IP,
		"ssh_user":    sw.SSHUser,
		"ssh_pass":    sw.SSHPass,
		"ssh_port":    sw.SSHPort,
		"vendor":      sw.Vendor,
		"port_name":   port.PortName,
		"vlan_id":     port.VlanID,
		"speed_mbps":  port.SpeedMbps,
		"admin_status": port.AdminStatus,
		"description": port.Description,
	}

	resp, err := s.hub.SendRequest(sw.AgentID, ws.ActionSwitchProvision, payload, 30*time.Second)
	if err != nil {
		return fmt.Errorf("agent request failed: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("agent error: %s", resp.Error)
	}
	return nil
}

// GetPortStatus queries live port status from the switch via the agent.
func (s *SwitchService) GetPortStatus(ctx context.Context, switchID uuid.UUID, portID uuid.UUID) (map[string]any, error) {
	sw, err := s.switchRepo.GetByID(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("switch not found: %w", err)
	}
	port, err := s.portRepo.GetByID(ctx, portID)
	if err != nil {
		return nil, fmt.Errorf("port not found: %w", err)
	}

	payload := map[string]any{
		"switch_ip":      sw.IP,
		"ssh_user":       sw.SSHUser,
		"ssh_pass":       sw.SSHPass,
		"ssh_port":       sw.SSHPort,
		"vendor":         sw.Vendor,
		"snmp_community": sw.SNMPCommunity,
		"snmp_version":   sw.SNMPVersion,
		"port_name":      port.PortName,
		"port_index":     port.PortIndex,
	}

	resp, err := s.hub.SendRequest(sw.AgentID, ws.ActionSwitchStatus, payload, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("agent request failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}

	result := make(map[string]any)
	raw, _ := json.Marshal(resp.Payload)
	json.Unmarshal(raw, &result)
	return result, nil
}

// TestConnection tests SSH and SNMP connectivity to a switch via the agent.
func (s *SwitchService) TestConnection(ctx context.Context, switchID uuid.UUID) (map[string]any, error) {
	sw, err := s.switchRepo.GetByID(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("switch not found: %w", err)
	}

	payload := map[string]any{
		"switch_ip":      sw.IP,
		"ssh_user":       sw.SSHUser,
		"ssh_pass":       sw.SSHPass,
		"ssh_port":       sw.SSHPort,
		"vendor":         sw.Vendor,
		"snmp_community": sw.SNMPCommunity,
		"snmp_version":   sw.SNMPVersion,
	}

	resp, err := s.hub.SendRequest(sw.AgentID, ws.ActionSwitchTest, payload, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("agent request failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}

	result := make(map[string]any)
	raw, _ := json.Marshal(resp.Payload)
	json.Unmarshal(raw, &result)
	return result, nil
}

// PollSNMP triggers SNMP polling on a switch and returns discovered port data.
func (s *SwitchService) PollSNMP(ctx context.Context, switchID uuid.UUID) (map[string]any, error) {
	sw, err := s.switchRepo.GetByID(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("switch not found: %w", err)
	}

	payload := map[string]any{
		"switch_ip":      sw.IP,
		"snmp_community": sw.SNMPCommunity,
		"snmp_version":   sw.SNMPVersion,
	}

	resp, err := s.hub.SendRequest(sw.AgentID, ws.ActionSNMPPoll, payload, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("agent request failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}

	result := make(map[string]any)
	raw, _ := json.Marshal(resp.Payload)
	json.Unmarshal(raw, &result)
	return result, nil
}

// SyncPortsFromSNMP polls SNMP for port data and upserts into the database.
func (s *SwitchService) SyncPortsFromSNMP(ctx context.Context, switchID uuid.UUID) ([]domain.SwitchPort, error) {
	sw, err := s.switchRepo.GetByID(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("switch not found: %w", err)
	}

	payload := map[string]any{
		"switch_ip":      sw.IP,
		"snmp_community": sw.SNMPCommunity,
		"snmp_version":   sw.SNMPVersion,
	}

	resp, err := s.hub.SendRequest(sw.AgentID, ws.ActionSNMPPoll, payload, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("agent request failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}

	// Parse response payload
	raw, _ := json.Marshal(resp.Payload)
	var pollResult struct {
		Ports []struct {
			PortIndex  int    `json:"port_index"`
			PortName   string `json:"port_name"`
			Speed      uint64 `json:"speed"`
			OperStatus string `json:"oper_status"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(raw, &pollResult); err != nil {
		return nil, fmt.Errorf("parse SNMP response: %w", err)
	}

	now := time.Now()
	for _, p := range pollResult.Ports {
		speedMbps := int(p.Speed / 1_000_000) // SNMP returns bits/sec
		port := &domain.SwitchPort{
			ID:          uuid.New(),
			SwitchID:    switchID,
			PortIndex:   p.PortIndex,
			PortName:    p.PortName,
			SpeedMbps:   speedMbps,
			AdminStatus: "up",
			OperStatus:  p.OperStatus,
			LastPolled:  &now,
		}
		if err := s.portRepo.UpsertBySwitchAndIndex(ctx, port); err != nil {
			s.logger.Warn("failed to upsert port from SNMP",
				zap.Int("port_index", p.PortIndex),
				zap.Error(err),
			)
		}
	}

	return s.portRepo.ListBySwitchID(ctx, switchID)
}

// StartPeriodicSNMPSync runs SNMP port polling for all switches at the given interval.
func (s *SwitchService) StartPeriodicSNMPSync(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Info("starting periodic SNMP port sync", zap.Duration("interval", interval))
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping periodic SNMP port sync")
			return
		case <-ticker.C:
			s.syncAllSwitchPorts(ctx)
		}
	}
}

func (s *SwitchService) syncAllSwitchPorts(ctx context.Context) {
	switches, err := s.switchRepo.List(ctx)
	if err != nil {
		s.logger.Error("periodic SNMP sync: failed to list switches", zap.Error(err))
		return
	}
	for _, sw := range switches {
		if _, err := s.SyncPortsFromSNMP(ctx, sw.ID); err != nil {
			s.logger.Debug("periodic SNMP sync failed for switch",
				zap.String("switch", sw.Name),
				zap.String("ip", sw.IP),
				zap.Error(err),
			)
		}
	}
}

// ConfigureDHCPRelay sends a DHCP relay configuration command to a switch via the agent.
func (s *SwitchService) ConfigureDHCPRelay(ctx context.Context, switchID uuid.UUID, req *domain.DHCPRelayRequest) error {
	sw, err := s.switchRepo.GetByID(ctx, switchID)
	if err != nil {
		return fmt.Errorf("switch not found: %w", err)
	}

	payload := map[string]any{
		"switch_ip":      sw.IP,
		"ssh_user":       sw.SSHUser,
		"ssh_pass":       sw.SSHPass,
		"ssh_port":       sw.SSHPort,
		"vendor":         sw.Vendor,
		"interface_name": req.InterfaceName,
		"dhcp_server_ip": req.DHCPServerIP,
		"relay_group":    req.RelayGroup,
		"remove":         req.Remove,
	}

	resp, err := s.hub.SendRequest(sw.AgentID, ws.ActionSwitchDHCPRelay, payload, 30*time.Second)
	if err != nil {
		return fmt.Errorf("agent request failed: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("agent error: %s", resp.Error)
	}
	return nil
}
