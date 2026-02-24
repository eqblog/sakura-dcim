package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/agent/internal/config"
)

// Message matches the panel's WebSocket message format
type Message struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"` // request, response, event
	Action  string      `json:"action"`
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ActionHandler processes incoming requests from the panel
type ActionHandler func(payload json.RawMessage) (interface{}, error)

// WSClient manages the WebSocket connection to the control panel
type WSClient struct {
	cfg      *config.Config
	logger   *zap.Logger
	conn     *websocket.Conn
	mu       sync.Mutex
	handlers map[string]ActionHandler
	done     chan struct{}
	startTime time.Time
}

func NewWSClient(cfg *config.Config, logger *zap.Logger, handlers map[string]ActionHandler) *WSClient {
	return &WSClient{
		cfg:       cfg,
		logger:    logger,
		handlers:  handlers,
		done:      make(chan struct{}),
		startTime: time.Now(),
	}
}

// ConnectWithRetry attempts to connect with exponential backoff
func (c *WSClient) ConnectWithRetry() {
	backoff := time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-c.done:
			return
		default:
		}

		err := c.connect()
		if err != nil {
			c.logger.Error("connection failed", zap.Error(err), zap.Duration("retry_in", backoff))
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff on successful connection
		backoff = time.Second
		c.readLoop()
	}
}

func (c *WSClient) connect() error {
	u, err := url.Parse(c.cfg.ServerURL)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	q.Set("agent_id", c.cfg.AgentID)
	q.Set("token", c.cfg.Token)
	u.RawQuery = q.Encode()

	c.logger.Info("connecting to panel", zap.String("url", u.Host))

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Start heartbeat
	go c.heartbeatLoop()

	c.logger.Info("connected to panel")
	return nil
}

func (c *WSClient) readLoop() {
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("ws read error", zap.Error(err))
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Error("unmarshal error", zap.Error(err))
			continue
		}

		if msg.Type == "request" {
			go c.handleRequest(&msg)
		}
	}
}

func (c *WSClient) handleRequest(msg *Message) {
	handler, ok := c.handlers[msg.Action]
	if !ok {
		c.sendResponse(msg.ID, nil, fmt.Sprintf("unknown action: %s", msg.Action))
		return
	}

	payloadBytes, _ := json.Marshal(msg.Payload)
	result, err := handler(payloadBytes)
	if err != nil {
		c.sendResponse(msg.ID, nil, err.Error())
		return
	}

	c.sendResponse(msg.ID, result, "")
}

func (c *WSClient) sendResponse(id string, payload interface{}, errMsg string) {
	resp := Message{
		ID:      id,
		Type:    "response",
		Payload: payload,
		Error:   errMsg,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		c.logger.Error("marshal response error", zap.Error(err))
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			c.logger.Error("write error", zap.Error(err))
		}
	}
}

func (c *WSClient) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hostname, _ := os.Hostname()
			heartbeat := Message{
				ID:     fmt.Sprintf("hb-%d", time.Now().UnixNano()),
				Type:   "event",
				Action: "agent.heartbeat",
				Payload: map[string]interface{}{
					"version":  "0.1.0",
					"uptime":   int64(time.Since(c.startTime).Seconds()),
					"hostname": hostname,
					"os":       runtime.GOOS,
					"arch":     runtime.GOARCH,
				},
			}

			data, _ := json.Marshal(heartbeat)

			c.mu.Lock()
			if c.conn != nil {
				c.conn.WriteMessage(websocket.TextMessage, data)
			}
			c.mu.Unlock()

		case <-c.done:
			return
		}
	}
}

func (c *WSClient) Close() {
	close(c.done)
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
}
