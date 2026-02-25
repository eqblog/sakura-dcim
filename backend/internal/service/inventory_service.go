package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

type InventoryService struct {
	inventoryRepo repository.InventoryRepository
	serverRepo    repository.ServerRepository
	hub           *ws.Hub
	cfg           *config.Config
	logger        *zap.Logger
}

func NewInventoryService(
	inventoryRepo repository.InventoryRepository,
	serverRepo repository.ServerRepository,
	hub *ws.Hub,
	cfg *config.Config,
	logger *zap.Logger,
) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
		serverRepo:    serverRepo,
		hub:           hub,
		cfg:           cfg,
		logger:        logger,
	}
}

// GetInventory returns all stored inventory components for a server.
func (s *InventoryService) GetInventory(ctx context.Context, serverID uuid.UUID) (*domain.InventoryResult, error) {
	components, err := s.inventoryRepo.ListByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	result := &domain.InventoryResult{
		ServerID:   serverID,
		Components: components,
	}
	if len(components) > 0 {
		t := components[0].CollectedAt
		result.CollectedAt = &t
	}
	return result, nil
}

// TriggerScan PXE-boots the target server into a mini-Linux for hardware scanning.
// This is asynchronous — the server reboots, scans hardware, and reports back via
// the agent callback. Results arrive as inventory.result events.
func (s *InventoryService) TriggerScan(ctx context.Context, serverID uuid.UUID) (map[string]interface{}, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	if server.AgentID == nil {
		return nil, fmt.Errorf("server has no agent assigned")
	}
	if server.MACAddress == "" {
		return nil, fmt.Errorf("server has no MAC address — set it in server settings before scanning")
	}

	// Decrypt IPMI credentials
	ipmiUser, ipmiPass := "", ""
	if server.IPMIUser != "" {
		ipmiUser, err = crypto.DecryptAESGCM(server.IPMIUser, s.cfg.Crypto.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt ipmi_user: %w", err)
		}
	}
	if server.IPMIPass != "" {
		ipmiPass, err = crypto.DecryptAESGCM(server.IPMIPass, s.cfg.Crypto.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt ipmi_pass: %w", err)
		}
	}

	// Send inventory.pxe to agent — this PXE boots the server into inventory mode
	resp, err := s.hub.SendRequest(*server.AgentID, ws.ActionInventoryPXE, map[string]any{
		"server_id":  serverID.String(),
		"ipmi_ip":    server.IPMIIP,
		"ipmi_user":  ipmiUser,
		"ipmi_pass":  ipmiPass,
		"server_mac": server.MACAddress,
		"bmc_type":   string(server.BMCType),
	}, 120*time.Second)
	if err != nil {
		return nil, fmt.Errorf("agent request failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}

	// Return immediately — PXE boot is async, results come via inventory.result event
	raw, _ := json.Marshal(resp.Payload)
	var result map[string]interface{}
	json.Unmarshal(raw, &result)
	return result, nil
}

// HandleInventoryResultEvent processes unsolicited inventory.result events from agents.
func (s *InventoryService) HandleInventoryResultEvent(agentID uuid.UUID, msg *ws.Message) {
	raw, _ := json.Marshal(msg.Payload)
	var payload struct {
		ServerID string                 `json:"server_id"`
		Data     map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		s.logger.Error("failed to parse inventory result event", zap.Error(err))
		return
	}

	serverID, err := uuid.Parse(payload.ServerID)
	if err != nil {
		s.logger.Error("invalid server_id in inventory result", zap.String("raw", payload.ServerID))
		return
	}

	ctx := context.Background()
	for component, details := range payload.Data {
		inv := &domain.ServerInventory{
			ServerID:  serverID,
			Component: component,
			Details:   details,
		}
		if err := s.inventoryRepo.Upsert(ctx, inv); err != nil {
			s.logger.Error("failed to store inventory from event",
				zap.String("component", component),
				zap.Error(err))
		}
	}

	s.logger.Info("inventory result stored",
		zap.String("server_id", serverID.String()),
		zap.Int("components", len(payload.Data)))
}
