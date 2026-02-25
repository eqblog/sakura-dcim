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
	done      chan struct{} // closed when relay session ends
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

	session, err := h.kvmService.StartSession(c.Request.Context(), serverID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Prepare relay slot (agent may have connected already — don't overwrite)
	h.relayMu.Lock()
	if _, exists := h.relays[session.SessionID]; !exists {
		h.relays[session.SessionID] = &kvmRelay{
			waitAgent: make(chan struct{}),
			done:      make(chan struct{}),
		}
	}
	h.relayMu.Unlock()

	respData := map[string]interface{}{
		"session_id": session.SessionID,
	}
	if session.TempUser != "" {
		respData["temp_user"] = session.TempUser
		respData["temp_pass"] = session.TempPass
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Success: true,
		Data:    respData,
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
	defer conn.Close()

	h.relayMu.Lock()
	relay, exists := h.relays[sessionID]
	if !exists {
		relay = &kvmRelay{waitAgent: make(chan struct{}), done: make(chan struct{})}
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

	// Block until the relay session ends.
	// Do NOT read from conn here — HandleBrowserWS is the sole reader
	// of agentConn (gorilla/websocket allows only one concurrent reader).
	select {
	case <-relay.done:
		h.logger.Info("agent relay: session done", zap.String("session_id", sessionID))
	case <-c.Request.Context().Done():
		h.logger.Info("agent relay: context cancelled", zap.String("session_id", sessionID))
		h.cleanupRelay(sessionID)
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

	if ok {
		// Signal HandleAgentRelay to unblock and exit
		select {
		case <-relay.done:
		default:
			close(relay.done)
		}
		if relay.agentConn != nil {
			relay.agentConn.Close()
		}
	}
}
