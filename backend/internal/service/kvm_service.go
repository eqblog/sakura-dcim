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
	SessionID string    `json:"session_id"`
	ServerID  uuid.UUID `json:"server_id"`
	AgentID   uuid.UUID `json:"agent_id"`
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token"`
	Status    string    `json:"status"` // starting, active, closing
	CreatedAt time.Time `json:"created_at"`
}

type KVMService struct {
	serverRepo repository.ServerRepository
	hub        *ws.Hub
	cfg        *config.Config
	logger     *zap.Logger
	sessions   map[string]*KVMSession
	mu         sync.RWMutex
}

func NewKVMService(serverRepo repository.ServerRepository, hub *ws.Hub, cfg *config.Config, logger *zap.Logger) *KVMService {
	svc := &KVMService{
		serverRepo: serverRepo,
		hub:        hub,
		cfg:        cfg,
		logger:     logger,
		sessions:   make(map[string]*KVMSession),
	}
	go svc.cleanupLoop()
	return svc
}

func (s *KVMService) StartSession(ctx context.Context, serverID, userID uuid.UUID, panelBaseURL string) (*KVMSession, error) {
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
		SessionID: sessionID,
		ServerID:  serverID,
		AgentID:   *server.AgentID,
		UserID:    userID,
		Token:     sessionToken,
		Status:    "starting",
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	// Build relay URL for agent to connect back
	relayURL := fmt.Sprintf("%s/api/v1/kvm/relay?session=%s&token=%s", panelBaseURL, sessionID, sessionToken)

	// Send KVM start command to agent
	payload := ws.KVMStartPayload{
		IPMIIP:   server.IPMIIP,
		IPMIUser: ipmiUser,
		IPMIPass: ipmiPass,
	}
	// Extend payload with session info
	fullPayload := map[string]interface{}{
		"ipmi_ip":    payload.IPMIIP,
		"ipmi_user":  payload.IPMIUser,
		"ipmi_pass":  payload.IPMIPass,
		"session_id": sessionID,
		"relay_url":  relayURL,
	}

	_, err = s.hub.SendRequest(*server.AgentID, ws.ActionIPMIKVMStart, fullPayload, 60*time.Second)
	if err != nil {
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

	stopPayload := map[string]string{"session_id": sessionID}
	_, err := s.hub.SendRequest(session.AgentID, ws.ActionIPMIKVMStop, stopPayload, 10*time.Second)
	if err != nil {
		s.logger.Warn("failed to send KVM stop to agent", zap.Error(err))
	}

	s.logger.Info("KVM session stopped", zap.String("session_id", sessionID))
	return nil
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
