package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
)

// ── message types (mirror backend/internal/websocket/protocol.go) ────

type Message struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Action  string `json:"action,omitempty"`
	Payload any    `json:"payload,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ── colors ───────────────────────────────────────────────────────────

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorRed    = "\033[31m"
	colorGray   = "\033[90m"
)

func logRecv(action string) {
	fmt.Printf("%s[RECV]%s %s%s%s\n", colorCyan, colorReset, colorYellow, action, colorReset)
}

func logSend(kind, detail string) {
	fmt.Printf("%s[SEND]%s %s %s\n", colorGreen, colorReset, kind, detail)
}

func logEvent(action string) {
	fmt.Printf("%s[EVNT]%s %s%s%s\n", colorCyan, colorReset, colorGray, action, colorReset)
}

func logError(msg string, err error) {
	fmt.Printf("%s[ERR]%s  %s: %v\n", colorRed, colorReset, msg, err)
}

// ── mock agent ───────────────────────────────────────────────────────

type MockAgent struct {
	conn       *gorillaws.Conn
	writeMu    sync.Mutex
	powerState string // "on" or "off"
	serverID   string // populated from pxe.prepare payload
}

func (a *MockAgent) sendJSON(msg *Message) error {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	return a.conn.WriteJSON(msg)
}

func (a *MockAgent) handleRequest(msg *Message) {
	logRecv(msg.Action)

	payload := a.dispatch(msg)

	resp := &Message{
		ID:      msg.ID,
		Type:    "response",
		Payload: payload,
	}
	if err := a.sendJSON(resp); err != nil {
		logError("send response", err)
		return
	}
	logSend("response", fmt.Sprintf("action=%s", msg.Action))
}

func (a *MockAgent) dispatch(msg *Message) any {
	switch msg.Action {

	// ── IPMI Power ───────────────────────────────────────────────
	case "ipmi.power.status":
		return map[string]any{"status": a.powerState}

	case "ipmi.power.on":
		a.powerState = "on"
		return map[string]any{"output": "Chassis Power Control: Up/On"}

	case "ipmi.power.off":
		a.powerState = "off"
		return map[string]any{"output": "Chassis Power Control: Down/Off"}

	case "ipmi.power.reset":
		a.powerState = "on"
		return map[string]any{"output": "Chassis Power Control: Reset"}

	case "ipmi.power.cycle":
		a.powerState = "on"
		return map[string]any{"output": "Chassis Power Control: Cycle"}

	// ── IPMI Sensors ─────────────────────────────────────────────
	case "ipmi.sensors":
		return map[string]any{
			"sensors": []map[string]any{
				{"name": "CPU1 Temp", "type": "temperature", "value": 42.0 + rand.Float64()*8, "unit": "C"},
				{"name": "CPU2 Temp", "type": "temperature", "value": 40.0 + rand.Float64()*8, "unit": "C"},
				{"name": "System Temp", "type": "temperature", "value": 28.0 + rand.Float64()*5, "unit": "C"},
				{"name": "FAN1", "type": "fan", "value": 3200 + rand.Float64()*800, "unit": "RPM"},
				{"name": "FAN2", "type": "fan", "value": 3100 + rand.Float64()*800, "unit": "RPM"},
				{"name": "FAN3", "type": "fan", "value": 3300 + rand.Float64()*800, "unit": "RPM"},
				{"name": "FAN4", "type": "fan", "value": 3150 + rand.Float64()*800, "unit": "RPM"},
				{"name": "12V", "type": "voltage", "value": 12.05 + rand.Float64()*0.1, "unit": "V"},
				{"name": "5V", "type": "voltage", "value": 5.02 + rand.Float64()*0.05, "unit": "V"},
				{"name": "3.3V", "type": "voltage", "value": 3.31 + rand.Float64()*0.03, "unit": "V"},
				{"name": "PS1 Input Power", "type": "power", "value": 180 + rand.Float64()*40, "unit": "W"},
				{"name": "PS2 Input Power", "type": "power", "value": 175 + rand.Float64()*40, "unit": "W"},
			},
		}

	// ── KVM ──────────────────────────────────────────────────────
	case "ipmi.kvm.start":
		sessionID := ""
		if raw, ok := msg.Payload.(map[string]any); ok {
			if v, ok := raw["session_id"].(string); ok {
				sessionID = v
			}
		}
		if sessionID == "" {
			sessionID = uuid.New().String()
		}
		return map[string]any{"session_id": sessionID, "port": 5900}

	case "ipmi.kvm.stop":
		return map[string]any{}

	// ── PXE Install ──────────────────────────────────────────────
	case "pxe.prepare":
		// Extract server_id from payload for async events
		serverID := ""
		if raw, ok := msg.Payload.(map[string]any); ok {
			if v, ok := raw["server_id"].(string); ok {
				serverID = v
			}
		}
		go a.simulatePXEInstall(serverID)
		return map[string]any{"status": "ok"}

	case "pxe.cleanup":
		return map[string]any{"status": "ok"}

	// ── Inventory ────────────────────────────────────────────────
	case "inventory.scan":
		serverID := ""
		if raw, ok := msg.Payload.(map[string]any); ok {
			if v, ok := raw["server_id"].(string); ok {
				serverID = v
			}
		}
		go a.simulateInventoryResult(serverID)
		return map[string]any{"status": "ok"}

	// ── Switch ───────────────────────────────────────────────────
	case "switch.provision":
		return map[string]any{"status": "ok"}

	case "switch.status":
		return map[string]any{
			"admin_status": "up",
			"oper_status":  "up",
			"speed_mbps":   10000,
			"vlan_id":      100,
		}

	// ── SNMP ─────────────────────────────────────────────────────
	case "snmp.poll":
		return map[string]any{
			"rx_bytes": 1024 * 1024 * rand.Int63n(1000),
			"tx_bytes": 1024 * 1024 * rand.Int63n(500),
		}

	// ── RAID ─────────────────────────────────────────────────────
	case "raid.configure":
		return map[string]any{"status": "ok"}

	case "raid.status":
		return map[string]any{"status": "ok", "level": "raid1", "state": "optimal"}

	default:
		log.Printf("unhandled action: %s", msg.Action)
		return map[string]any{}
	}
}

// simulatePXEInstall sends a sequence of pxe.status events over ~6 seconds.
func (a *MockAgent) simulatePXEInstall(serverID string) {
	steps := []struct {
		delay    time.Duration
		status   string
		progress int
		message  string
	}{
		{0, "pxe_booting", 10, "PXE environment prepared. Server booting from network..."},
		{2 * time.Second, "installing", 40, "Installing base system..."},
		{2 * time.Second, "installing", 70, "Installing packages and kernel..."},
		{1 * time.Second, "post_scripts", 90, "Running post-installation scripts..."},
		{1 * time.Second, "completed", 100, "Installation completed successfully."},
	}

	for _, step := range steps {
		time.Sleep(step.delay)
		event := &Message{
			ID:     uuid.New().String(),
			Type:   "event",
			Action: "pxe.status",
			Payload: map[string]any{
				"server_id": serverID,
				"status":    step.status,
				"progress":  step.progress,
				"message":   step.message,
			},
		}
		if err := a.sendJSON(event); err != nil {
			logError("send pxe.status event", err)
			return
		}
		logEvent(fmt.Sprintf("pxe.status → %s (%d%%)", step.status, step.progress))
	}
}

// simulateInventoryResult sends an inventory.result event with realistic hardware data.
func (a *MockAgent) simulateInventoryResult(serverID string) {
	time.Sleep(500 * time.Millisecond)
	event := &Message{
		ID:     uuid.New().String(),
		Type:   "event",
		Action: "inventory.result",
		Payload: map[string]any{
			"server_id": serverID,
			"components": map[string]any{
				"cpu": map[string]any{
					"model":   "Intel Xeon E-2288G @ 3.70GHz",
					"cores":   8,
					"threads": 16,
					"sockets": 1,
				},
				"memory": map[string]any{
					"total_mb": 65536,
					"modules": []map[string]any{
						{"slot": "DIMM_A1", "size_mb": 16384, "type": "DDR4", "speed_mhz": 2666},
						{"slot": "DIMM_A2", "size_mb": 16384, "type": "DDR4", "speed_mhz": 2666},
						{"slot": "DIMM_B1", "size_mb": 16384, "type": "DDR4", "speed_mhz": 2666},
						{"slot": "DIMM_B2", "size_mb": 16384, "type": "DDR4", "speed_mhz": 2666},
					},
				},
				"disks": []map[string]any{
					{"slot": "0", "model": "Samsung SSD 870 EVO", "serial": "S5Y0NF0R123456", "size_bytes": 500107862016, "type": "ssd"},
					{"slot": "1", "model": "Samsung SSD 870 EVO", "serial": "S5Y0NF0R654321", "size_bytes": 500107862016, "type": "ssd"},
				},
				"network": []map[string]any{
					{"name": "eth0", "mac": "00:25:90:f1:a2:b3", "speed_mbps": 10000, "driver": "ixgbe"},
					{"name": "eth1", "mac": "00:25:90:f1:a2:b4", "speed_mbps": 10000, "driver": "ixgbe"},
					{"name": "ipmi", "mac": "00:25:90:f1:a2:b5", "speed_mbps": 1000, "driver": ""},
				},
				"system": map[string]any{
					"manufacturer": "Supermicro",
					"product":      "SYS-5019C-MR",
					"serial":       "A331234567890",
					"bios_version": "3.4",
				},
			},
		},
	}
	if err := a.sendJSON(event); err != nil {
		logError("send inventory.result event", err)
		return
	}
	logEvent("inventory.result → components sent")
}

// heartbeatLoop sends agent.heartbeat events every 30s.
func (a *MockAgent) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	startTime := time.Now()

	hostname, _ := os.Hostname()

	for range ticker.C {
		event := &Message{
			ID:     fmt.Sprintf("hb-%d", time.Now().UnixNano()),
			Type:   "event",
			Action: "agent.heartbeat",
			Payload: map[string]any{
				"version":  "mock-0.1.0",
				"uptime":   int(time.Since(startTime).Seconds()),
				"hostname": hostname,
				"os":       "linux",
				"arch":     "amd64",
			},
		}
		if err := a.sendJSON(event); err != nil {
			return
		}
		logEvent("agent.heartbeat")
	}
}

// readLoop reads messages from the WebSocket and dispatches requests.
func (a *MockAgent) readLoop() {
	for {
		_, data, err := a.conn.ReadMessage()
		if err != nil {
			logError("read", err)
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			logError("unmarshal", err)
			continue
		}

		if msg.Type == "request" {
			go a.handleRequest(&msg)
		}
	}
}

// ── main ─────────────────────────────────────────────────────────────

func main() {
	serverURL := flag.String("server", "ws://localhost:8080/api/v1/agents/ws", "Backend WebSocket URL")
	agentID := flag.String("id", "", "Agent UUID (from API creation)")
	agentToken := flag.String("token", "", "Agent token (from API creation)")
	flag.Parse()

	if *agentID == "" || *agentToken == "" {
		fmt.Println("Usage: mock-agent -id <agent-uuid> -token <agent-token> [-server <ws-url>]")
		os.Exit(1)
	}

	// Build connection URL
	u, err := url.Parse(*serverURL)
	if err != nil {
		log.Fatalf("invalid server URL: %v", err)
	}
	q := u.Query()
	q.Set("agent_id", *agentID)
	q.Set("token", *agentToken)
	u.RawQuery = q.Encode()

	fmt.Printf("\n%s╔══════════════════════════════════════════╗%s\n", colorGreen, colorReset)
	fmt.Printf("%s║      Sakura DCIM Mock Agent v0.1.0       ║%s\n", colorGreen, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════╝%s\n\n", colorGreen, colorReset)
	fmt.Printf("Agent ID:  %s\n", *agentID)
	fmt.Printf("Server:    %s\n", *serverURL)
	fmt.Printf("Connecting...\n\n")

	// Connect
	dialer := gorillaws.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("%s[FAIL]%s WebSocket connection failed: %v\n", colorRed, colorReset, err)
	}
	defer conn.Close()

	fmt.Printf("%s[OK]%s   Connected to backend!\n\n", colorGreen, colorReset)

	agent := &MockAgent{
		conn:       conn,
		powerState: "on",
	}

	// Start heartbeat
	go agent.heartbeatLoop()

	// Handle graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go func() {
		<-interrupt
		fmt.Printf("\n%sShutting down mock agent...%s\n", colorYellow, colorReset)
		conn.WriteMessage(gorillaws.CloseMessage,
			gorillaws.FormatCloseMessage(gorillaws.CloseNormalClosure, ""))
		os.Exit(0)
	}()

	// Read loop (blocks until connection closed)
	agent.readLoop()
}
