package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

type DiscoveryService struct {
	sessionRepo    repository.DiscoverySessionRepository
	discoveredRepo repository.DiscoveredServerRepository
	serverRepo     repository.ServerRepository
	agentRepo      repository.AgentRepository
	hub            *ws.Hub
	logger         *zap.Logger
}

func NewDiscoveryService(
	sessionRepo repository.DiscoverySessionRepository,
	discoveredRepo repository.DiscoveredServerRepository,
	serverRepo repository.ServerRepository,
	agentRepo repository.AgentRepository,
	hub *ws.Hub,
	logger *zap.Logger,
) *DiscoveryService {
	return &DiscoveryService{
		sessionRepo:    sessionRepo,
		discoveredRepo: discoveredRepo,
		serverRepo:     serverRepo,
		agentRepo:      agentRepo,
		hub:            hub,
		logger:         logger,
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// StartDiscovery initiates a discovery session on an agent.
func (s *DiscoveryService) StartDiscovery(ctx context.Context, agentID uuid.UUID, userID *uuid.UUID, req *domain.DiscoveryStartRequest) (*domain.DiscoverySession, error) {
	// Verify agent exists
	if _, err := s.agentRepo.GetByID(ctx, agentID); err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	// Check for existing active session
	existing, _ := s.sessionRepo.GetActiveByAgentID(ctx, agentID)
	if existing != nil {
		return nil, fmt.Errorf("agent already has an active discovery session")
	}

	// Check agent is online
	if !s.hub.IsAgentOnline(agentID) {
		return nil, fmt.Errorf("agent is not online")
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	dhcpRange := fmt.Sprintf("%s-%s", req.DHCPRangeStart, req.DHCPRangeEnd)

	session := &domain.DiscoverySession{
		AgentID:       agentID,
		Status:        domain.DiscoveryStatusActive,
		CallbackToken: token,
		DHCPRange:     dhcpRange,
		StartedBy:     userID,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Send discovery.start to agent (fire-and-forget with timeout)
	go func() {
		_, err := s.hub.SendRequest(agentID, ws.ActionDiscoveryStart, map[string]any{
			"session_id":       session.ID.String(),
			"callback_token":   token,
			"dhcp_range_start": req.DHCPRangeStart,
			"dhcp_range_end":   req.DHCPRangeEnd,
			"gateway":          req.Gateway,
			"netmask":          req.Netmask,
			"interface":        req.Interface,
		}, 30*time.Second)
		if err != nil {
			s.logger.Error("failed to send discovery.start to agent",
				zap.String("agent_id", agentID.String()),
				zap.Error(err))
		}
	}()

	return session, nil
}

// StopDiscovery stops an active discovery session on an agent.
func (s *DiscoveryService) StopDiscovery(ctx context.Context, agentID uuid.UUID) error {
	session, err := s.sessionRepo.GetActiveByAgentID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("no active discovery session: %w", err)
	}

	if err := s.sessionRepo.UpdateStatus(ctx, session.ID, domain.DiscoveryStatusStopped); err != nil {
		return fmt.Errorf("update session status: %w", err)
	}

	// Send stop to agent
	go func() {
		_, err := s.hub.SendRequest(agentID, ws.ActionDiscoveryStop, map[string]any{
			"session_id": session.ID.String(),
		}, 15*time.Second)
		if err != nil {
			s.logger.Error("failed to send discovery.stop to agent",
				zap.String("agent_id", agentID.String()),
				zap.Error(err))
		}
	}()

	return nil
}

// GetDiscoveryStatus returns the active session info and discovered count.
func (s *DiscoveryService) GetDiscoveryStatus(ctx context.Context, agentID uuid.UUID) (map[string]any, error) {
	session, err := s.sessionRepo.GetActiveByAgentID(ctx, agentID)
	if err != nil {
		return map[string]any{"active": false}, nil
	}

	list, err := s.discoveredRepo.List(ctx, domain.DiscoveredServerListParams{
		AgentID:  &agentID,
		Page:     1,
		PageSize: 1,
	})
	count := int64(0)
	if err == nil && list != nil {
		count = list.Total
	}

	return map[string]any{
		"active":           true,
		"session_id":       session.ID,
		"dhcp_range":       session.DHCPRange,
		"started_at":       session.StartedAt,
		"discovered_count": count,
	}, nil
}

// HandleDiscoveryResultEvent processes discovery.result events from agents.
func (s *DiscoveryService) HandleDiscoveryResultEvent(agentID uuid.UUID, msg *ws.Message) {
	raw, _ := json.Marshal(msg.Payload)
	var payload struct {
		SessionID    string `json:"session_id"`
		MACAddress   string `json:"mac_address"`
		IPAddress    string `json:"ip_address"`
		System       struct {
			Vendor  string `json:"vendor"`
			Product string `json:"product"`
			Serial  string `json:"serial"`
		} `json:"system"`
		CPU struct {
			Model   string `json:"model"`
			Cores   int    `json:"cores"`
			Sockets int    `json:"sockets"`
		} `json:"cpu"`
		Memory struct {
			TotalMB int64 `json:"total_mb"`
		} `json:"memory"`
		DiskSummary struct {
			Count   int   `json:"count"`
			TotalGB int64 `json:"total_gb"`
		} `json:"disk_summary"`
		NICCount int    `json:"nic_count"`
		BMC      struct {
			IP string `json:"ip"`
		} `json:"bmc"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		s.logger.Error("failed to parse discovery result", zap.Error(err))
		return
	}

	sessionID, err := uuid.Parse(payload.SessionID)
	if err != nil {
		s.logger.Error("invalid session_id in discovery result", zap.String("raw", payload.SessionID))
		return
	}

	ctx := context.Background()

	ds := &domain.DiscoveredServer{
		SessionID:     sessionID,
		AgentID:       agentID,
		MACAddress:    payload.MACAddress,
		IPAddress:     payload.IPAddress,
		Status:        domain.DiscoveredStatusPending,
		SystemVendor:  payload.System.Vendor,
		SystemProduct: payload.System.Product,
		SystemSerial:  payload.System.Serial,
		CPUModel:      payload.CPU.Model,
		CPUCores:      payload.CPU.Cores,
		CPUSockets:    payload.CPU.Sockets,
		RAMMB:         payload.Memory.TotalMB,
		DiskCount:     payload.DiskSummary.Count,
		DiskTotalGB:   payload.DiskSummary.TotalGB,
		NICCount:      payload.NICCount,
		RawInventory:  msg.Payload,
		BMCIP:         payload.BMC.IP,
	}

	if err := s.discoveredRepo.Upsert(ctx, ds); err != nil {
		s.logger.Error("failed to upsert discovered server",
			zap.String("mac", payload.MACAddress),
			zap.Error(err))
	}
}

// ListDiscoveredServers returns a paginated list of discovered servers.
func (s *DiscoveryService) ListDiscoveredServers(ctx context.Context, params domain.DiscoveredServerListParams) (*domain.PaginatedResult[domain.DiscoveredServer], error) {
	return s.discoveredRepo.List(ctx, params)
}

// GetDiscoveredServer returns a single discovered server by ID.
func (s *DiscoveryService) GetDiscoveredServer(ctx context.Context, id uuid.UUID) (*domain.DiscoveredServer, error) {
	return s.discoveredRepo.GetByID(ctx, id)
}

// ApproveServer creates a managed Server from a discovered server.
func (s *DiscoveryService) ApproveServer(ctx context.Context, id uuid.UUID, userID *uuid.UUID, req *domain.DiscoveryApproveRequest) (*domain.Server, error) {
	ds, err := s.discoveredRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("discovered server not found: %w", err)
	}
	if ds.Status != domain.DiscoveredStatusPending {
		return nil, fmt.Errorf("server already %s", ds.Status)
	}

	agentID := &ds.AgentID
	if req.AgentID != nil {
		agentID = req.AgentID
	}

	// Auto-detect BMC type from vendor if not specified
	bmcType := req.BMCType
	if bmcType == "" {
		bmcType = domain.DetectBMCType(ds.SystemVendor)
	}

	server := &domain.Server{
		AgentID:    agentID,
		Hostname:   req.Hostname,
		Label:      req.Label,
		Status:     domain.ServerStatusActive,
		IPMIIP:     req.IPMIIP,
		IPMIUser:   req.IPMIUser,
		IPMIPass:   req.IPMIPass,
		MACAddress: ds.MACAddress,
		BMCType:    bmcType,
		CPUModel:   ds.CPUModel,
		CPUCores:   ds.CPUCores,
		RAMMB:      ds.RAMMB,
		Tags:       req.Tags,
		Notes:      req.Notes,
	}

	if err := s.serverRepo.Create(ctx, server); err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}

	// Mark discovered server as approved
	if err := s.discoveredRepo.UpdateStatus(ctx, id, domain.DiscoveredStatusApproved); err != nil {
		s.logger.Error("failed to update discovered server status", zap.Error(err))
	}
	if err := s.discoveredRepo.SetServerID(ctx, id, server.ID); err != nil {
		s.logger.Error("failed to set server_id on discovered server", zap.Error(err))
	}

	return server, nil
}

// RejectServer marks a discovered server as rejected.
func (s *DiscoveryService) RejectServer(ctx context.Context, id uuid.UUID) error {
	ds, err := s.discoveredRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("discovered server not found: %w", err)
	}
	if ds.Status != domain.DiscoveredStatusPending {
		return fmt.Errorf("server already %s", ds.Status)
	}
	return s.discoveredRepo.UpdateStatus(ctx, id, domain.DiscoveredStatusRejected)
}

// DeleteDiscoveredServer removes a discovered server record.
func (s *DiscoveryService) DeleteDiscoveredServer(ctx context.Context, id uuid.UUID) error {
	return s.discoveredRepo.Delete(ctx, id)
}
