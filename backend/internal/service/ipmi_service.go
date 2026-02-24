package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

// IPMIService handles power control and sensor queries via agent IPMI.
type IPMIService struct {
	serverRepo repository.ServerRepository
	hub        *ws.Hub
	cfg        *config.Config
	logger     *zap.Logger
}

func NewIPMIService(serverRepo repository.ServerRepository, hub *ws.Hub, cfg *config.Config, logger *zap.Logger) *IPMIService {
	return &IPMIService{
		serverRepo: serverRepo,
		hub:        hub,
		cfg:        cfg,
		logger:     logger,
	}
}

// powerActionMap maps user-facing action names to WebSocket action constants.
var powerActionMap = map[string]string{
	"on":     ws.ActionIPMIPowerOn,
	"off":    ws.ActionIPMIPowerOff,
	"reset":  ws.ActionIPMIPowerReset,
	"cycle":  ws.ActionIPMIPowerCycle,
	"status": ws.ActionIPMIPowerStatus,
}

// decryptIPMI fetches the server and returns decrypted IPMI credentials + agentID.
func (s *IPMIService) decryptIPMI(ctx context.Context, serverID uuid.UUID) (agentID uuid.UUID, ipmiIP, ipmiUser, ipmiPass string, err error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return uuid.Nil, "", "", "", fmt.Errorf("server not found: %w", err)
	}

	if server.AgentID == nil {
		return uuid.Nil, "", "", "", fmt.Errorf("server has no agent assigned")
	}

	if server.IPMIIP == "" {
		return uuid.Nil, "", "", "", fmt.Errorf("server has no IPMI IP configured")
	}

	if !s.hub.IsAgentOnline(*server.AgentID) {
		return uuid.Nil, "", "", "", fmt.Errorf("agent is offline")
	}

	if server.IPMIUser != "" {
		ipmiUser, err = crypto.DecryptAESGCM(server.IPMIUser, s.cfg.Crypto.EncryptionKey)
		if err != nil {
			return uuid.Nil, "", "", "", fmt.Errorf("decrypt ipmi_user: %w", err)
		}
	}
	if server.IPMIPass != "" {
		ipmiPass, err = crypto.DecryptAESGCM(server.IPMIPass, s.cfg.Crypto.EncryptionKey)
		if err != nil {
			return uuid.Nil, "", "", "", fmt.Errorf("decrypt ipmi_pass: %w", err)
		}
	}

	return *server.AgentID, server.IPMIIP, ipmiUser, ipmiPass, nil
}

// PowerAction executes a power control command (on/off/reset/cycle).
func (s *IPMIService) PowerAction(ctx context.Context, serverID uuid.UUID, action string) (map[string]interface{}, error) {
	wsAction, ok := powerActionMap[action]
	if !ok {
		return nil, fmt.Errorf("invalid power action: %s (valid: on, off, reset, cycle, status)", action)
	}

	agentID, ipmiIP, ipmiUser, ipmiPass, err := s.decryptIPMI(ctx, serverID)
	if err != nil {
		return nil, err
	}

	payload := ws.PowerPayload{
		IPMIIP:   ipmiIP,
		IPMIUser: ipmiUser,
		IPMIPass: ipmiPass,
	}

	s.logger.Info("IPMI power action",
		zap.String("server_id", serverID.String()),
		zap.String("action", action),
	)

	resp, err := s.hub.SendRequest(agentID, wsAction, payload, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("agent power %s failed: %w", action, err)
	}

	// Parse response payload into map
	result := make(map[string]interface{})
	if resp.Payload != nil {
		raw, err := json.Marshal(resp.Payload)
		if err == nil {
			json.Unmarshal(raw, &result)
		}
	}

	return result, nil
}

// GetPowerStatus queries the current power state of a server.
func (s *IPMIService) GetPowerStatus(ctx context.Context, serverID uuid.UUID) (string, error) {
	agentID, ipmiIP, ipmiUser, ipmiPass, err := s.decryptIPMI(ctx, serverID)
	if err != nil {
		return "unknown", err
	}

	payload := ws.PowerPayload{
		IPMIIP:   ipmiIP,
		IPMIUser: ipmiUser,
		IPMIPass: ipmiPass,
	}

	resp, err := s.hub.SendRequest(agentID, ws.ActionIPMIPowerStatus, payload, 15*time.Second)
	if err != nil {
		return "unknown", fmt.Errorf("agent power status failed: %w", err)
	}

	// Extract status from response
	if resp.Payload != nil {
		raw, err := json.Marshal(resp.Payload)
		if err == nil {
			var result map[string]interface{}
			if json.Unmarshal(raw, &result) == nil {
				if s, ok := result["status"].(string); ok {
					return s, nil
				}
			}
		}
	}

	return "unknown", nil
}

// GetSensors queries IPMI sensor data from the agent.
func (s *IPMIService) GetSensors(ctx context.Context, serverID uuid.UUID) ([]map[string]string, error) {
	agentID, ipmiIP, ipmiUser, ipmiPass, err := s.decryptIPMI(ctx, serverID)
	if err != nil {
		return nil, err
	}

	payload := ws.PowerPayload{
		IPMIIP:   ipmiIP,
		IPMIUser: ipmiUser,
		IPMIPass: ipmiPass,
	}

	resp, err := s.hub.SendRequest(agentID, ws.ActionIPMISensors, payload, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("agent sensors query failed: %w", err)
	}

	// Parse response payload
	if resp.Payload != nil {
		raw, err := json.Marshal(resp.Payload)
		if err == nil {
			var result struct {
				Sensors []map[string]string `json:"sensors"`
			}
			if json.Unmarshal(raw, &result) == nil {
				return result.Sensors, nil
			}
		}
	}

	return nil, nil
}
