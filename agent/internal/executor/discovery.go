package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"text/template"
	"time"

	"go.uber.org/zap"
)

// WSEventSender allows the discovery executor to send events back to the panel.
type WSEventSender interface {
	SendEvent(action string, payload interface{})
}

// DiscoveryExecutor handles PXE-based hardware discovery.
type DiscoveryExecutor struct {
	logger      *zap.Logger
	wsClient    WSEventSender
	mu          sync.Mutex
	activeToken string
	sessionID   string
	httpServer  *http.Server
}

func NewDiscoveryExecutor(logger *zap.Logger, wsClient WSEventSender) *DiscoveryExecutor {
	return &DiscoveryExecutor{
		logger:   logger,
		wsClient: wsClient,
	}
}

type discoveryStartPayload struct {
	SessionID      string `json:"session_id"`
	CallbackToken  string `json:"callback_token"`
	DHCPRangeStart string `json:"dhcp_range_start"`
	DHCPRangeEnd   string `json:"dhcp_range_end"`
	Gateway        string `json:"gateway"`
	Netmask        string `json:"netmask"`
	Interface      string `json:"interface"`
}

// HandleDiscoveryStart sets up dnsmasq for PXE discovery and starts the HTTP callback server.
func (e *DiscoveryExecutor) HandleDiscoveryStart(payload json.RawMessage) (interface{}, error) {
	var p discoveryStartPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	e.mu.Lock()
	if e.activeToken != "" {
		e.mu.Unlock()
		return nil, fmt.Errorf("discovery already active")
	}
	e.activeToken = p.CallbackToken
	e.sessionID = p.SessionID
	e.mu.Unlock()

	// Write dnsmasq discovery config
	if err := e.writeDnsmasqConfig(&p); err != nil {
		e.cleanup()
		return nil, fmt.Errorf("write dnsmasq config: %w", err)
	}

	// Write PXE boot config pointing to discovery kernel
	if err := e.writePXEConfig(&p); err != nil {
		e.cleanup()
		return nil, fmt.Errorf("write pxe config: %w", err)
	}

	// Reload dnsmasq
	if err := exec.Command("systemctl", "reload", "dnsmasq").Run(); err != nil {
		e.logger.Warn("dnsmasq reload failed, trying restart", zap.Error(err))
		if err := exec.Command("systemctl", "restart", "dnsmasq").Run(); err != nil {
			e.cleanup()
			return nil, fmt.Errorf("restart dnsmasq: %w", err)
		}
	}

	// Start HTTP callback server on :9877
	if err := e.startHTTPServer(); err != nil {
		e.cleanup()
		return nil, fmt.Errorf("start http server: %w", err)
	}

	e.logger.Info("discovery started",
		zap.String("session_id", p.SessionID),
		zap.String("dhcp_range", p.DHCPRangeStart+"-"+p.DHCPRangeEnd))

	return map[string]string{"status": "started"}, nil
}

// HandleDiscoveryStop tears down the discovery environment.
func (e *DiscoveryExecutor) HandleDiscoveryStop(payload json.RawMessage) (interface{}, error) {
	e.cleanup()
	e.logger.Info("discovery stopped")
	return map[string]string{"status": "stopped"}, nil
}

func (e *DiscoveryExecutor) cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.activeToken = ""
	e.sessionID = ""

	// Stop HTTP server
	if e.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		e.httpServer.Shutdown(ctx)
		e.httpServer = nil
	}

	// Remove dnsmasq discovery config
	os.Remove("/etc/dnsmasq.d/discovery.conf")

	// Remove PXE discovery config
	os.Remove("/srv/tftp/pxelinux.cfg/discovery")

	// Reload dnsmasq
	exec.Command("systemctl", "reload", "dnsmasq").Run()
}

var dnsmasqTmpl = template.Must(template.New("dnsmasq").Parse(`# Sakura DCIM Discovery Mode — auto-generated
{{- if .Interface}}
interface={{.Interface}}
bind-interfaces
{{- end}}
dhcp-range={{.DHCPRangeStart}},{{.DHCPRangeEnd}},{{.Netmask}},1h
dhcp-option=3,{{.Gateway}}
dhcp-boot=pxelinux.0
enable-tftp
tftp-root=/srv/tftp
`))

func (e *DiscoveryExecutor) writeDnsmasqConfig(p *discoveryStartPayload) error {
	f, err := os.Create("/etc/dnsmasq.d/discovery.conf")
	if err != nil {
		return err
	}
	defer f.Close()
	return dnsmasqTmpl.Execute(f, p)
}

// getAgentIP returns the IP address of the agent host for callback URL.
func getAgentIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

var pxeCfgTmpl = template.Must(template.New("pxe").Parse(`DEFAULT discovery
LABEL discovery
    KERNEL vmlinuz-discovery
    APPEND initrd=initrd-discovery.img sakura.callback=http://{{.AgentIP}}:9877/discovery/report sakura.token={{.Token}} sakura.session={{.SessionID}}
`))

func (e *DiscoveryExecutor) writePXEConfig(p *discoveryStartPayload) error {
	// Ensure directory exists
	os.MkdirAll("/srv/tftp/pxelinux.cfg", 0755)

	f, err := os.Create("/srv/tftp/pxelinux.cfg/default")
	if err != nil {
		return err
	}
	defer f.Close()

	return pxeCfgTmpl.Execute(f, map[string]string{
		"AgentIP":   getAgentIP(),
		"Token":     p.CallbackToken,
		"SessionID": p.SessionID,
	})
}

func (e *DiscoveryExecutor) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/discovery/report", e.handleReport)

	e.mu.Lock()
	e.httpServer = &http.Server{
		Addr:         ":9877",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	e.mu.Unlock()

	listener, err := net.Listen("tcp", ":9877")
	if err != nil {
		return fmt.Errorf("listen :9877: %w", err)
	}

	go func() {
		if err := e.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			e.logger.Error("discovery http server error", zap.Error(err))
		}
	}()

	return nil
}

// handleReport receives hardware inventory from PXE-booted servers.
func (e *DiscoveryExecutor) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	e.mu.Lock()
	activeToken := e.activeToken
	sessionID := e.sessionID
	e.mu.Unlock()

	if token == "" || token != activeToken {
		http.Error(w, "invalid token", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		http.Error(w, "read body error", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var report map[string]interface{}
	if err := json.Unmarshal(body, &report); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Attach session_id to the report
	report["session_id"] = sessionID

	e.logger.Info("received discovery report",
		zap.String("session_id", sessionID),
		zap.Any("mac", report["mac_address"]))

	// Forward to panel via WebSocket event
	e.wsClient.SendEvent("discovery.result", report)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
