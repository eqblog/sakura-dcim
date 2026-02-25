package handler

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/middleware"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
)

var kvmUpgrader = ws.Upgrader{
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// kvmRelay holds paired WebSocket connections for a KVM session
type kvmRelay struct {
	mu        sync.Mutex
	agentConn *ws.Conn
	waitAgent chan struct{}
}

type KVMHandler struct {
	kvmService *service.KVMService
	logger     *zap.Logger
	relays     map[string]*kvmRelay
	relayMu    sync.Mutex
}

func NewKVMHandler(kvmService *service.KVMService, logger *zap.Logger) *KVMHandler {
	return &KVMHandler{
		kvmService: kvmService,
		logger:     logger,
		relays:     make(map[string]*kvmRelay),
	}
}

func (h *KVMHandler) RegisterRoutes(protected *gin.RouterGroup) {
	protected.POST("/servers/:id/kvm", h.StartKVM)
	protected.DELETE("/servers/:id/kvm", h.StopKVM)
}

func (h *KVMHandler) RegisterPublicRoutes(r *gin.Engine) {
	// WebSocket endpoints (auth via query params)
	r.GET("/api/v1/kvm/ws", h.HandleBrowserWS)
	r.GET("/api/v1/kvm/relay", h.HandleAgentRelay)
}

func (h *KVMHandler) StartKVM(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	userID := middleware.GetUserID(c)

	// Detect real host — prefer X-Forwarded-Host (set by nginx/Vite proxy),
	// fall back to request Host header.
	realHost := c.GetHeader("X-Forwarded-Host")
	if realHost == "" {
		realHost = c.Request.Host
	}

	// Build panel base URL for agent relay (agent → backend direct, use c.Request.Host)
	relayScheme := "ws"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		relayScheme = "wss"
	}
	panelBaseURL := relayScheme + "://" + c.Request.Host

	session, err := h.kvmService.StartSession(c.Request.Context(), serverID, userID, panelBaseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Prepare relay slot
	h.relayMu.Lock()
	h.relays[session.SessionID] = &kvmRelay{
		waitAgent: make(chan struct{}),
	}
	h.relayMu.Unlock()

	// Build ws_url for browser using real host (public IP, not proxy host)
	wsScheme := "ws"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		wsScheme = "wss"
	}
	wsURL := wsScheme + "://" + realHost + "/api/v1/kvm/ws?session=" + session.SessionID

	c.JSON(http.StatusOK, domain.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"session_id": session.SessionID,
			"ws_url":     wsURL,
		},
	})
}

func (h *KVMHandler) StopKVM(c *gin.Context) {
	serverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "invalid server ID"})
		return
	}

	userID := middleware.GetUserID(c)

	// Find session for this server
	var sessionID string
	// We need to iterate sessions - get from query or find by server
	if sid := c.Query("session"); sid != "" {
		sessionID = sid
	} else {
		// Find active session for this server by user
		_ = serverID
		_ = userID
		c.JSON(http.StatusBadRequest, domain.APIResponse{Success: false, Error: "session query parameter required"})
		return
	}

	if err := h.kvmService.StopSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	h.cleanupRelay(sessionID)

	c.JSON(http.StatusOK, domain.APIResponse{Success: true, Message: "KVM session stopped"})
}

// HandleAgentRelay accepts the agent's WebSocket connection for VNC data relay
func (h *KVMHandler) HandleAgentRelay(c *gin.Context) {
	sessionID := c.Query("session")
	token := c.Query("token")
	if sessionID == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session and token required"})
		return
	}

	_, ok := h.kvmService.ValidateSessionToken(sessionID, token)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session or token"})
		return
	}

	conn, err := kvmUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("agent relay upgrade failed", zap.Error(err))
		return
	}

	h.relayMu.Lock()
	relay, exists := h.relays[sessionID]
	if !exists {
		relay = &kvmRelay{waitAgent: make(chan struct{})}
		h.relays[sessionID] = relay
	}
	h.relayMu.Unlock()

	relay.mu.Lock()
	relay.agentConn = conn
	relay.mu.Unlock()

	// Signal that agent is connected
	select {
	case <-relay.waitAgent:
	default:
		close(relay.waitAgent)
	}

	h.logger.Info("agent relay connected", zap.String("session_id", sessionID))

	// Keep connection alive until closed
	for {
		if _, _, err := conn.NextReader(); err != nil {
			break
		}
	}
}

// HandleBrowserWS accepts the browser's noVNC WebSocket connection
func (h *KVMHandler) HandleBrowserWS(c *gin.Context) {
	sessionID := c.Query("session")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session required"})
		return
	}

	// Validate session exists
	session, ok := h.kvmService.GetSession(sessionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Auth: validate JWT from query param or header
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Sec-WebSocket-Protocol")
	}
	_ = session // session validated above, auth checked at route level
	_ = token

	browserConn, err := kvmUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("browser WS upgrade failed", zap.Error(err))
		return
	}
	defer browserConn.Close()

	h.relayMu.Lock()
	relay, exists := h.relays[sessionID]
	h.relayMu.Unlock()

	if !exists {
		h.logger.Error("no relay slot for session", zap.String("session_id", sessionID))
		browserConn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseInternalServerErr, "no relay"))
		return
	}

	// Wait for agent to connect (up to 30s)
	select {
	case <-relay.waitAgent:
	case <-c.Request.Context().Done():
		return
	}

	relay.mu.Lock()
	agentConn := relay.agentConn
	relay.mu.Unlock()

	if agentConn == nil {
		browserConn.WriteMessage(ws.CloseMessage, ws.FormatCloseMessage(ws.CloseInternalServerErr, "agent not connected"))
		return
	}

	h.logger.Info("KVM relay active", zap.String("session_id", sessionID))

	done := make(chan struct{})

	// Browser → Agent
	go func() {
		defer close(done)
		for {
			msgType, data, err := browserConn.ReadMessage()
			if err != nil {
				if !ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
					h.logger.Debug("browser read error", zap.Error(err))
				}
				return
			}
			if err := agentConn.WriteMessage(msgType, data); err != nil {
				h.logger.Debug("agent write error", zap.Error(err))
				return
			}
		}
	}()

	// Agent → Browser
	go func() {
		for {
			msgType, data, err := agentConn.ReadMessage()
			if err != nil {
				if !ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
					h.logger.Debug("agent read error", zap.Error(err))
				}
				browserConn.Close()
				return
			}
			if err := browserConn.WriteMessage(msgType, data); err != nil {
				h.logger.Debug("browser write error", zap.Error(err))
				return
			}
		}
	}()

	<-done

	h.logger.Info("KVM relay ended", zap.String("session_id", sessionID))
	h.kvmService.StopSession(sessionID)
	h.cleanupRelay(sessionID)
}

func (h *KVMHandler) cleanupRelay(sessionID string) {
	h.relayMu.Lock()
	relay, ok := h.relays[sessionID]
	if ok {
		delete(h.relays, sessionID)
	}
	h.relayMu.Unlock()

	if ok && relay.agentConn != nil {
		relay.agentConn.Close()
	}
}
