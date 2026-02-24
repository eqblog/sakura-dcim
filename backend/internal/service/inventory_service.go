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

type InventoryService struct {
	inventoryRepo repository.InventoryRepository
	serverRepo    repository.ServerRepository
	hub           *ws.Hub
	logger        *zap.Logger
}

func NewInventoryService(
	inventoryRepo repository.InventoryRepository,
	serverRepo repository.ServerRepository,
	hub *ws.Hub,
	logger *zap.Logger,
) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
		serverRepo:    serverRepo,
		hub:           hub,
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

// TriggerScan sends an inventory.scan request to the agent and stores the results.
func (s *InventoryService) TriggerScan(ctx context.Context, serverID uuid.UUID) (*domain.InventoryResult, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	if server.AgentID == nil {
		return nil, fmt.Errorf("server has no agent assigned")
	}

	resp, err := s.hub.SendRequest(*server.AgentID, ws.ActionInventoryScan, map[string]any{
		"server_id": serverID.String(),
	}, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("agent request failed: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}

	// Parse agent response — it returns a map of component → details
	raw, _ := json.Marshal(resp.Payload)
	var scanResult map[string]interface{}
	if err := json.Unmarshal(raw, &scanResult); err != nil {
		return nil, fmt.Errorf("failed to parse scan result: %w", err)
	}

	// Store each component
	for component, details := range scanResult {
		inv := &domain.ServerInventory{
			ServerID:  serverID,
			Component: component,
			Details:   details,
		}
		if err := s.inventoryRepo.Upsert(ctx, inv); err != nil {
			s.logger.Error("failed to store inventory component",
				zap.String("component", component),
				zap.Error(err))
		}
	}

	return s.GetInventory(ctx, serverID)
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
}
