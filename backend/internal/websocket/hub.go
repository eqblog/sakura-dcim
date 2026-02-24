package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// AgentConnection represents a connected agent
type AgentConnection struct {
	AgentID  uuid.UUID
	Conn     *ws.Conn
	Send     chan []byte
	hub      *Hub
	logger   *zap.Logger

	// Pending requests waiting for responses
	pendingMu sync.Mutex
	pending   map[string]chan *Message
}

// Hub manages all agent WebSocket connections
type Hub struct {
	agents     map[uuid.UUID]*AgentConnection
	mu         sync.RWMutex
	logger     *zap.Logger
	register   chan *AgentConnection
	unregister chan *AgentConnection
}

// NewHub creates a new WebSocket hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		agents:     make(map[uuid.UUID]*AgentConnection),
		logger:     logger,
		register:   make(chan *AgentConnection),
		unregister: make(chan *AgentConnection),
	}
}

// Run starts the hub event loop
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			// Close existing connection for same agent
			if old, ok := h.agents[conn.AgentID]; ok {
				close(old.Send)
				old.Conn.Close()
			}
			h.agents[conn.AgentID] = conn
			h.mu.Unlock()
			h.logger.Info("agent connected", zap.String("agent_id", conn.AgentID.String()))

		case conn := <-h.unregister:
			h.mu.Lock()
			if existing, ok := h.agents[conn.AgentID]; ok && existing == conn {
				delete(h.agents, conn.AgentID)
				close(conn.Send)
			}
			h.mu.Unlock()
			h.logger.Info("agent disconnected", zap.String("agent_id", conn.AgentID.String()))
		}
	}
}

// Register adds an agent connection to the hub
func (h *Hub) Register(conn *AgentConnection) {
	h.register <- conn
}

// Unregister removes an agent connection from the hub
func (h *Hub) Unregister(conn *AgentConnection) {
	h.unregister <- conn
}

// GetAgent returns the connection for a specific agent
func (h *Hub) GetAgent(agentID uuid.UUID) (*AgentConnection, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conn, ok := h.agents[agentID]
	return conn, ok
}

// IsAgentOnline checks if an agent is connected
func (h *Hub) IsAgentOnline(agentID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.agents[agentID]
	return ok
}

// OnlineAgentIDs returns a list of currently connected agent IDs
func (h *Hub) OnlineAgentIDs() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]uuid.UUID, 0, len(h.agents))
	for id := range h.agents {
		ids = append(ids, id)
	}
	return ids
}

// SendRequest sends a request to an agent and waits for the response
func (h *Hub) SendRequest(agentID uuid.UUID, action string, payload any, timeout time.Duration) (*Message, error) {
	conn, ok := h.GetAgent(agentID)
	if !ok {
		return nil, fmt.Errorf("agent %s is not connected", agentID)
	}

	msg := NewRequest(action, payload)

	// Set up response channel
	respCh := make(chan *Message, 1)
	conn.pendingMu.Lock()
	conn.pending[msg.ID] = respCh
	conn.pendingMu.Unlock()

	defer func() {
		conn.pendingMu.Lock()
		delete(conn.pending, msg.ID)
		conn.pendingMu.Unlock()
	}()

	// Send message
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
	}

	select {
	case conn.Send <- data:
	default:
		return nil, fmt.Errorf("agent %s send buffer full", agentID)
	}

	// Wait for response
	select {
	case resp := <-respCh:
		if resp.Error != "" {
			return nil, fmt.Errorf("agent error: %s", resp.Error)
		}
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("request to agent %s timed out", agentID)
	}
}

// NewAgentConnection creates a new agent connection
func NewAgentConnection(agentID uuid.UUID, conn *ws.Conn, hub *Hub, logger *zap.Logger) *AgentConnection {
	return &AgentConnection{
		AgentID: agentID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		hub:     hub,
		logger:  logger,
		pending: make(map[string]chan *Message),
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *AgentConnection) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseAbnormalClosure) {
				c.logger.Error("ws read error", zap.Error(err), zap.String("agent_id", c.AgentID.String()))
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Error("ws unmarshal error", zap.Error(err))
			continue
		}

		c.handleMessage(&msg)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *AgentConnection) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(ws.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *AgentConnection) handleMessage(msg *Message) {
	switch msg.Type {
	case TypeResponse:
		// Route response to pending request
		c.pendingMu.Lock()
		ch, ok := c.pending[msg.ID]
		c.pendingMu.Unlock()
		if ok {
			ch <- msg
		}

	case TypeEvent:
		// Handle agent-initiated events
		c.logger.Debug("agent event",
			zap.String("agent_id", c.AgentID.String()),
			zap.String("action", msg.Action),
		)
		// TODO: Route events to appropriate handlers (PXE status, SNMP data, etc.)
	}
}
