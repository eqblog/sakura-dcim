package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gorilla "github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

var upgrader = gorilla.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Agents connect from various IPs
	},
}

// HandleAgentWebSocket handles agent WebSocket connections
func HandleAgentWebSocket(c *gin.Context, hub *ws.Hub, agentRepo repository.AgentRepository, logger *zap.Logger) {
	// Authenticate agent via query parameter token
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	agentIDStr := c.Query("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	// Verify agent exists and token matches
	agent, err := agentRepo.GetByID(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "agent not found"})
		return
	}

	if !crypto.CheckPassword(token, agent.TokenHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("ws upgrade failed", zap.Error(err))
		return
	}

	agentConn := ws.NewAgentConnection(agentID, conn, hub, logger)
	hub.Register(agentConn)

	// Update agent status
	_ = agentRepo.UpdateLastSeen(c.Request.Context(), agentID)

	go agentConn.WritePump()
	go agentConn.ReadPump()
}
