package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// getHostGateway returns the address used to reach the Docker host.
// Defaults to "127.0.0.1" for native agent, set AGENT_HOST_GATEWAY=host.docker.internal
// when the agent itself runs inside a Docker container.
func getHostGateway() string {
	if gw := os.Getenv("AGENT_HOST_GATEWAY"); gw != "" {
		return gw
	}
	return "127.0.0.1"
}

const (
	kvmImageName       = "sakura-dcim/kvm-browser:latest"
	kvmContainerPrefix = "sakura-kvm-"
	kvmSessionTimeout  = 30 * time.Minute
)

type KVMStartPayload struct {
	IPMIIP    string `json:"ipmi_ip"`
	IPMIUser  string `json:"ipmi_user"`
	IPMIPass  string `json:"ipmi_pass"`
	BMCType   string `json:"bmc_type"`
	SessionID string `json:"session_id"`
	RelayURL  string `json:"relay_url"`
}

type KVMStopPayload struct {
	SessionID string `json:"session_id"`
}

type kvmSession struct {
	sessionID   string
	containerID string
	vncPort     string
	cancel      context.CancelFunc
	startedAt   time.Time
}

type KVMExecutor struct {
	logger   *zap.Logger
	sessions map[string]*kvmSession
	mu       sync.Mutex
}

func NewKVMExecutor(logger *zap.Logger) *KVMExecutor {
	return &KVMExecutor{
		logger:   logger,
		sessions: make(map[string]*kvmSession),
	}
}

func (e *KVMExecutor) HandleKVMStart(raw json.RawMessage) (interface{}, error) {
	var p KVMStartPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if p.IPMIIP == "" || p.SessionID == "" || p.RelayURL == "" {
		return nil, fmt.Errorf("ipmi_ip, session_id, relay_url are required")
	}

	e.mu.Lock()
	if _, exists := e.sessions[p.SessionID]; exists {
		e.mu.Unlock()
		return nil, fmt.Errorf("session %s already exists", p.SessionID)
	}
	e.mu.Unlock()

	targetURL := buildKVMTargetURL(p.BMCType, p.IPMIIP)
	containerName := kvmContainerPrefix + p.SessionID
	gateway := getHostGateway()

	// When agent runs inside Docker, bind to all interfaces so the host port is reachable
	portBind := "127.0.0.1"
	if gateway != "127.0.0.1" {
		portBind = "0.0.0.0"
	}

	// Start Docker container
	args := []string{
		"run", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("%s::5900", portBind),
		"-e", fmt.Sprintf("TARGET_URL=%s", targetURL),
		"-e", "SCREEN_WIDTH=1280",
		"-e", "SCREEN_HEIGHT=1024",
		"--memory=512m",
		"--cpus=1",
		kvmImageName,
	}

	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker run failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	containerID := strings.TrimSpace(string(out))
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	// Get the mapped VNC port
	portOut, err := exec.Command("docker", "port", containerName, "5900").CombinedOutput()
	if err != nil {
		exec.Command("docker", "rm", "-f", containerName).Run()
		return nil, fmt.Errorf("docker port failed: %s: %w", strings.TrimSpace(string(portOut)), err)
	}

	// Parse port output: "127.0.0.1:xxxxx"
	vncPort := parseDockerPort(strings.TrimSpace(string(portOut)))
	if vncPort == "" {
		exec.Command("docker", "rm", "-f", containerName).Run()
		return nil, fmt.Errorf("failed to parse VNC port from: %s", string(portOut))
	}

	// Wait for VNC to be ready
	if err := waitForTCP(gateway+":"+vncPort, 30*time.Second); err != nil {
		exec.Command("docker", "rm", "-f", containerName).Run()
		return nil, fmt.Errorf("VNC not ready: %w", err)
	}

	sessionCtx, cancel := context.WithCancel(context.Background())
	session := &kvmSession{
		sessionID:   p.SessionID,
		containerID: containerName,
		vncPort:     vncPort,
		cancel:      cancel,
		startedAt:   time.Now(),
	}

	e.mu.Lock()
	e.sessions[p.SessionID] = session
	e.mu.Unlock()

	// Rewrite relay URL if running inside Docker (127.0.0.1 → gateway)
	relayURL := p.RelayURL
	if gateway != "127.0.0.1" {
		relayURL = strings.ReplaceAll(relayURL, "127.0.0.1", gateway)
		relayURL = strings.ReplaceAll(relayURL, "localhost", gateway)
	}

	// Start VNC relay in background
	go e.relayVNC(sessionCtx, session, relayURL, gateway)

	e.logger.Info("KVM session started",
		zap.String("session_id", p.SessionID),
		zap.String("container", containerName),
		zap.String("vnc_port", vncPort),
		zap.String("target", p.IPMIIP),
	)

	return map[string]interface{}{
		"session_id": p.SessionID,
		"ready":      true,
	}, nil
}

func (e *KVMExecutor) HandleKVMStop(raw json.RawMessage) (interface{}, error) {
	var p KVMStopPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if p.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	e.stopSession(p.SessionID)
	return map[string]string{"status": "stopped"}, nil
}

func (e *KVMExecutor) stopSession(sessionID string) {
	e.mu.Lock()
	session, ok := e.sessions[sessionID]
	if !ok {
		e.mu.Unlock()
		return
	}
	delete(e.sessions, sessionID)
	e.mu.Unlock()

	session.cancel()
	exec.Command("docker", "rm", "-f", session.containerID).Run()
	e.logger.Info("KVM session stopped", zap.String("session_id", sessionID))
}

func (e *KVMExecutor) relayVNC(ctx context.Context, session *kvmSession, relayURL, gateway string) {
	defer e.stopSession(session.sessionID)

	u, err := url.Parse(relayURL)
	if err != nil {
		e.logger.Error("invalid relay URL", zap.Error(err))
		return
	}

	wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		e.logger.Error("failed to connect to panel relay", zap.Error(err), zap.String("url", u.String()))
		return
	}
	defer wsConn.Close()

	vncAddr := gateway + ":" + session.vncPort
	vncConn, err := net.DialTimeout("tcp", vncAddr, 5*time.Second)
	if err != nil {
		e.logger.Error("failed to connect to VNC", zap.Error(err), zap.String("addr", vncAddr))
		return
	}
	defer vncConn.Close()

	e.logger.Info("VNC relay active",
		zap.String("session_id", session.sessionID),
		zap.String("vnc_addr", vncAddr),
	)

	done := make(chan struct{})

	// VNC TCP → WebSocket
	go func() {
		defer close(done)
		buf := make([]byte, 32*1024)
		for {
			n, err := vncConn.Read(buf)
			if err != nil {
				if err != io.EOF {
					e.logger.Debug("VNC read error", zap.Error(err))
				}
				return
			}
			if err := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				e.logger.Debug("WS write error", zap.Error(err))
				return
			}
		}
	}()

	// WebSocket → VNC TCP
	go func() {
		for {
			_, data, err := wsConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					e.logger.Debug("WS read error", zap.Error(err))
				}
				vncConn.Close()
				return
			}
			if _, err := vncConn.Write(data); err != nil {
				e.logger.Debug("VNC write error", zap.Error(err))
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}

func (e *KVMExecutor) CleanupExpired() {
	e.mu.Lock()
	var expired []string
	for id, session := range e.sessions {
		if time.Since(session.startedAt) > kvmSessionTimeout {
			expired = append(expired, id)
		}
	}
	e.mu.Unlock()

	for _, id := range expired {
		e.logger.Info("KVM session expired", zap.String("session_id", id))
		e.stopSession(id)
	}
}

func (e *KVMExecutor) StopAll() {
	e.mu.Lock()
	ids := make([]string, 0, len(e.sessions))
	for id := range e.sessions {
		ids = append(ids, id)
	}
	e.mu.Unlock()

	for _, id := range ids {
		e.stopSession(id)
	}
}

func parseDockerPort(output string) string {
	// Output format: "127.0.0.1:12345" or "0.0.0.0:12345" or "[::]:12345\n127.0.0.1:12345"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

// buildKVMTargetURL returns the vendor-specific BMC web UI URL for Chromium kiosk mode.
func buildKVMTargetURL(bmcType, ip string) string {
	switch bmcType {
	case "dell_idrac":
		return fmt.Sprintf("https://%s/restgui/start.html", ip)
	case "hp_ilo":
		return fmt.Sprintf("https://%s/html/login.html", ip)
	case "supermicro":
		return fmt.Sprintf("https://%s/cgi/login.cgi", ip)
	case "lenovo_xcc":
		return fmt.Sprintf("https://%s/index.html", ip)
	case "huawei_ibmc":
		return fmt.Sprintf("https://%s/login.html", ip)
	default:
		return fmt.Sprintf("https://%s", ip)
	}
}

func waitForTCP(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}
