package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

type KVMSession struct {
	SessionID     string    `json:"session_id"`
	ServerID      uuid.UUID `json:"server_id"`
	AgentID       uuid.UUID `json:"agent_id"`
	UserID        uuid.UUID `json:"user_id"`
	Token         string    `json:"token"`
	Status        string    `json:"status"` // starting, active, closing
	CreatedAt     time.Time `json:"created_at"`
	TempUser      string    `json:"temp_user,omitempty"`
	TempPass      string    `json:"temp_pass,omitempty"`
	TempUserSlot  int       `json:"temp_user_slot,omitempty"`
	DirectConsole bool      `json:"direct_console,omitempty"` // auto-login + vConsole redirect
}

type KVMService struct {
	serverRepo repository.ServerRepository
	tenantRepo repository.TenantRepository
	hub        *ws.Hub
	cfg        *config.Config
	logger     *zap.Logger
	sessions   map[string]*KVMSession
	mu         sync.RWMutex
}

func NewKVMService(serverRepo repository.ServerRepository, tenantRepo repository.TenantRepository, hub *ws.Hub, cfg *config.Config, logger *zap.Logger) *KVMService {
	svc := &KVMService{
		serverRepo: serverRepo,
		tenantRepo: tenantRepo,
		hub:        hub,
		cfg:        cfg,
		logger:     logger,
		sessions:   make(map[string]*KVMSession),
	}
	go svc.cleanupLoop()
	return svc
}

func (s *KVMService) StartSession(ctx context.Context, serverID, userID, tenantID uuid.UUID) (*KVMSession, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	if server.AgentID == nil {
		return nil, fmt.Errorf("server has no agent assigned")
	}

	if server.IPMIIP == "" {
		return nil, fmt.Errorf("server has no IPMI IP configured")
	}

	if !s.hub.IsAgentOnline(*server.AgentID) {
		return nil, fmt.Errorf("agent is offline")
	}

	// Determine KVM mode from user's tenant setting
	directConsole := false
	if tenant, err := s.tenantRepo.GetByID(ctx, tenantID); err == nil {
		directConsole = tenant.KvmMode == "vconsole"
	}

	// Check for existing session on this server
	s.mu.RLock()
	for _, sess := range s.sessions {
		if sess.ServerID == serverID && sess.Status != "closing" {
			s.mu.RUnlock()
			return sess, nil
		}
	}
	s.mu.RUnlock()

	// Decrypt IPMI credentials
	ipmiUser := ""
	ipmiPass := ""
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

	sessionID := uuid.New().String()
	sessionToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}

	session := &KVMSession{
		SessionID:     sessionID,
		ServerID:      serverID,
		AgentID:       *server.AgentID,
		UserID:        userID,
		Token:         sessionToken,
		Status:        "starting",
		CreatedAt:     time.Now(),
		DirectConsole: directConsole,
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	// Build relay URL for agent to connect back.
	// Always use 127.0.0.1:<port> so the agent connects directly to the backend,
	// bypassing any reverse proxy (nginx). The agent rewrites 127.0.0.1 to
	// host.docker.internal when running inside Docker.
	relayURL := fmt.Sprintf("ws://127.0.0.1:%d/api/v1/kvm/relay?session=%s&token=%s",
		s.cfg.Server.Port, sessionID, sessionToken)

	// Create temporary IPMI user on the BMC for this session.
	// Non-fatal: if it fails, KVM still starts but without temp credentials.
	tempUserPayload := map[string]interface{}{
		"ipmi_ip":   server.IPMIIP,
		"ipmi_user": ipmiUser,
		"ipmi_pass": ipmiPass,
		"bmc_type":  string(server.BMCType),
		"privilege": 4, // Administrator — needed for BMC web console access
	}
	tempResp, tempErr := s.hub.SendRequest(*server.AgentID, ws.ActionIPMIUserCreate, tempUserPayload, 30*time.Second)
	if tempErr != nil {
		s.logger.Warn("failed to create temp IPMI user (KVM will start without it)",
			zap.String("session_id", sessionID),
			zap.Error(tempErr))
	} else if tempResp.Error != "" {
		s.logger.Warn("agent returned error creating temp IPMI user",
			zap.String("session_id", sessionID),
			zap.String("error", tempResp.Error))
	} else {
		// Extract temp credentials from agent response
		if respMap, ok := tempResp.Payload.(map[string]interface{}); ok {
			if u, ok := respMap["username"].(string); ok {
				session.TempUser = u
			}
			if p, ok := respMap["password"].(string); ok {
				session.TempPass = p
			}
			if slot, ok := respMap["user_slot"].(float64); ok {
				session.TempUserSlot = int(slot)
			}
			s.logger.Info("temp IPMI user created for KVM session",
				zap.String("session_id", sessionID),
				zap.String("temp_user", session.TempUser),
				zap.Int("slot", session.TempUserSlot))
		}
	}

	// Send KVM start command to agent
	fullPayload := map[string]interface{}{
		"ipmi_ip":        server.IPMIIP,
		"ipmi_user":      ipmiUser,
		"ipmi_pass":      ipmiPass,
		"bmc_type":       string(server.BMCType),
		"session_id":     sessionID,
		"relay_url":      relayURL,
		"direct_console": directConsole,
	}

	// For direct console: pass auto-login credentials so Docker container
	// can log into BMC automatically and redirect to vConsole.
	if directConsole {
		if session.TempUser != "" {
			fullPayload["auto_user"] = session.TempUser
			fullPayload["auto_pass"] = session.TempPass
		} else {
			// Fallback to admin credentials if temp user creation failed
			fullPayload["auto_user"] = ipmiUser
			fullPayload["auto_pass"] = ipmiPass
		}
	}

	// Direct console mode runs auto-login synchronously inside Docker before
	// starting x11vnc (so the user never sees the BMC login page).
	// Allow 120s for this: 90s VNC wait + agent + network overhead.
	// Web KVM mode starts x11vnc immediately, so 60s is plenty.
	kvmStartTimeout := 60 * time.Second
	if directConsole {
		kvmStartTimeout = 120 * time.Second
	}
	_, err = s.hub.SendRequest(*server.AgentID, ws.ActionIPMIKVMStart, fullPayload, kvmStartTimeout)
	if err != nil {
		// Cleanup temp user if we created one
		if session.TempUserSlot > 0 {
			s.deleteTempUser(*server.AgentID, server.IPMIIP, ipmiUser, ipmiPass, string(server.BMCType), session.TempUserSlot)
		}
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		return nil, fmt.Errorf("agent failed to start KVM: %w", err)
	}

	session.Status = "active"
	s.logger.Info("KVM session started",
		zap.String("session_id", sessionID),
		zap.String("server_id", serverID.String()),
	)

	return session, nil
}

func (s *KVMService) StopSession(sessionID string) error {
	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("session not found")
	}
	session.Status = "closing"
	delete(s.sessions, sessionID)
	s.mu.Unlock()

	// Delete temporary IPMI user if one was created
	if session.TempUserSlot > 0 {
		s.deleteTempUserForSession(session)
	}

	stopPayload := map[string]string{"session_id": sessionID}
	_, err := s.hub.SendRequest(session.AgentID, ws.ActionIPMIKVMStop, stopPayload, 10*time.Second)
	if err != nil {
		s.logger.Warn("failed to send KVM stop to agent", zap.Error(err))
	}

	s.logger.Info("KVM session stopped", zap.String("session_id", sessionID))
	return nil
}

// deleteTempUserForSession removes the temp IPMI user associated with a KVM session.
// It re-decrypts IPMI credentials from the database because we don't store them in session.
func (s *KVMService) deleteTempUserForSession(session *KVMSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	server, err := s.serverRepo.GetByID(ctx, session.ServerID)
	if err != nil {
		s.logger.Warn("cannot fetch server for temp user cleanup",
			zap.String("session_id", session.SessionID), zap.Error(err))
		return
	}

	ipmiUser, ipmiPass := "", ""
	if server.IPMIUser != "" {
		ipmiUser, _ = crypto.DecryptAESGCM(server.IPMIUser, s.cfg.Crypto.EncryptionKey)
	}
	if server.IPMIPass != "" {
		ipmiPass, _ = crypto.DecryptAESGCM(server.IPMIPass, s.cfg.Crypto.EncryptionKey)
	}

	s.deleteTempUser(session.AgentID, server.IPMIIP, ipmiUser, ipmiPass, string(server.BMCType), session.TempUserSlot)
}

// deleteTempUser sends ipmi.user.delete to the agent to remove a temporary IPMI user.
func (s *KVMService) deleteTempUser(agentID uuid.UUID, ipmiIP, ipmiUser, ipmiPass, bmcType string, userSlot int) {
	payload := map[string]interface{}{
		"ipmi_ip":   ipmiIP,
		"ipmi_user": ipmiUser,
		"ipmi_pass": ipmiPass,
		"bmc_type":  bmcType,
		"user_slot": userSlot,
	}
	resp, err := s.hub.SendRequest(agentID, ws.ActionIPMIUserDelete, payload, 15*time.Second)
	if err != nil {
		s.logger.Warn("failed to delete temp IPMI user",
			zap.Int("slot", userSlot), zap.Error(err))
	} else if resp.Error != "" {
		s.logger.Warn("agent error deleting temp IPMI user",
			zap.Int("slot", userSlot), zap.String("error", resp.Error))
	} else {
		s.logger.Info("temp IPMI user deleted", zap.Int("slot", userSlot))
	}
}

func (s *KVMService) GetSession(sessionID string) (*KVMSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID]
	return sess, ok
}

func (s *KVMService) ValidateSessionToken(sessionID, token string) (*KVMSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID]
	if !ok || sess.Token != token {
		return nil, false
	}
	return sess, true
}

func (s *KVMService) ValidateSessionUser(sessionID string, userID uuid.UUID) (*KVMSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return nil, false
	}
	return sess, true
}

func (s *KVMService) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		var expired []string
		for id, sess := range s.sessions {
			if time.Since(sess.CreatedAt) > 30*time.Minute {
				expired = append(expired, id)
			}
		}
		s.mu.Unlock()

		for _, id := range expired {
			s.logger.Info("KVM session expired", zap.String("session_id", id))
			s.StopSession(id)
		}
	}
}
