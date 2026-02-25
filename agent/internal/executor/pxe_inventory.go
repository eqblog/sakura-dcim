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
	"strings"
	"sync"
	"text/template"
	"time"

	"go.uber.org/zap"
)

// PXEInventoryExecutor boots a server to mini-Linux for hardware scanning
type PXEInventoryExecutor struct {
	logger   *zap.Logger
	ipmi     *IPMIExecutor
	pxe      *PXEExecutor
	wsClient WSEventSender

	mu         sync.Mutex
	httpServer *http.Server
	serverID   string
}

func NewPXEInventoryExecutor(logger *zap.Logger, ipmi *IPMIExecutor, pxe *PXEExecutor, wsClient WSEventSender) *PXEInventoryExecutor {
	return &PXEInventoryExecutor{logger: logger, ipmi: ipmi, pxe: pxe, wsClient: wsClient}
}

type PXEInventoryPayload struct {
	IPMIIP    string `json:"ipmi_ip"`
	IPMIUser  string `json:"ipmi_user"`
	IPMIPass  string `json:"ipmi_pass"`
	BMCType   string `json:"bmc_type"`
	ServerMAC string `json:"server_mac"`
	ServerID  string `json:"server_id"`
}

var invDnsmasqTmpl = template.Must(template.New("inv-dnsmasq").Parse(`# Sakura DCIM PXE Inventory — auto-generated for {{.ServerMAC}}
dhcp-host={{.ServerMAC}},set:inventory-{{.Tag}}
dhcp-boot=tag:inventory-{{.Tag}},pxelinux.0
enable-tftp
tftp-root=/srv/tftp
`))

var invPxeCfgTmpl = template.Must(template.New("inv-pxe").Parse(`DEFAULT inventory
LABEL inventory
    KERNEL vmlinuz-discovery
    APPEND initrd=initrd-discovery.img sakura.callback=http://{{.AgentIP}}:9878/inventory/report sakura.server_id={{.ServerID}} sakura.mode=inventory
`))

// HandlePXEInventory boots a server into inventory-mode mini-Linux, scans hardware, reports back.
func (e *PXEInventoryExecutor) HandlePXEInventory(raw json.RawMessage) (interface{}, error) {
	var p PXEInventoryPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse pxe inventory payload: %w", err)
	}

	if p.IPMIIP == "" || p.ServerMAC == "" || p.ServerID == "" {
		return nil, fmt.Errorf("ipmi_ip, server_mac, and server_id are required")
	}

	// Strip CIDR notation if present
	if idx := strings.IndexByte(p.IPMIIP, '/'); idx != -1 {
		p.IPMIIP = p.IPMIIP[:idx]
	}

	e.logger.Info("starting PXE inventory mode",
		zap.String("server_id", p.ServerID),
		zap.String("mac", p.ServerMAC),
	)

	// Use a short tag from server_id for dnsmasq tag naming
	tag := p.ServerID
	if len(tag) > 8 {
		tag = tag[:8]
	}

	// 1. Write dnsmasq config for this specific server MAC
	confPath := fmt.Sprintf("/etc/dnsmasq.d/pxe-inventory-%s.conf", tag)
	f, err := os.Create(confPath)
	if err != nil {
		return nil, fmt.Errorf("create dnsmasq config: %w", err)
	}
	if err := invDnsmasqTmpl.Execute(f, map[string]string{
		"ServerMAC": p.ServerMAC,
		"Tag":       tag,
	}); err != nil {
		f.Close()
		return nil, fmt.Errorf("write dnsmasq config: %w", err)
	}
	f.Close()

	// 2. Write PXE boot config pointing to inventory kernel
	os.MkdirAll("/srv/tftp/pxelinux.cfg", 0755)
	// Create a MAC-specific PXE config (01-xx-xx-xx-xx-xx-xx format)
	macFile := "01-" + strings.ReplaceAll(strings.ToLower(p.ServerMAC), ":", "-")
	pxeCfgPath := fmt.Sprintf("/srv/tftp/pxelinux.cfg/%s", macFile)
	pf, err := os.Create(pxeCfgPath)
	if err != nil {
		return nil, fmt.Errorf("create PXE config: %w", err)
	}
	if err := invPxeCfgTmpl.Execute(pf, map[string]string{
		"AgentIP":  getAgentIP(),
		"ServerID": p.ServerID,
	}); err != nil {
		pf.Close()
		return nil, fmt.Errorf("write PXE config: %w", err)
	}
	pf.Close()

	// 3. Reload dnsmasq
	if err := exec.Command("systemctl", "reload", "dnsmasq").Run(); err != nil {
		e.logger.Warn("dnsmasq reload failed, trying restart", zap.Error(err))
		exec.Command("systemctl", "restart", "dnsmasq").Run()
	}

	// 4. Start HTTP callback server on :9878 for inventory reports
	if err := e.startCallbackServer(p.ServerID); err != nil {
		e.cleanup(confPath, pxeCfgPath)
		return nil, fmt.Errorf("start callback server: %w", err)
	}

	// 5. Set IPMI to PXE boot and reboot the server
	if p.IPMIUser != "" && p.IPMIPass != "" {
		_, err := e.ipmi.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass,
			"chassis", "bootdev", "pxe")
		if err != nil {
			e.cleanup(confPath, pxeCfgPath)
			return nil, fmt.Errorf("set PXE boot: %w", err)
		}

		_, err = e.ipmi.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass,
			"chassis", "power", "cycle")
		if err != nil {
			e.logger.Warn("power cycle failed, trying power on", zap.Error(err))
			e.ipmi.runIPMI(p.BMCType, p.IPMIIP, p.IPMIUser, p.IPMIPass,
				"chassis", "power", "on")
		}
	}

	// 6. Schedule cleanup after timeout (10 minutes)
	go func() {
		time.Sleep(10 * time.Minute)
		e.cleanup(confPath, pxeCfgPath)
	}()

	e.logger.Info("PXE inventory boot initiated",
		zap.String("server_id", p.ServerID),
	)

	return map[string]interface{}{
		"status":    "booting",
		"server_id": p.ServerID,
		"message":   "Server is PXE booting into inventory mode. Hardware report will arrive automatically.",
	}, nil
}

func (e *PXEInventoryExecutor) startCallbackServer(serverID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop existing server if any
	if e.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		e.httpServer.Shutdown(ctx)
	}

	e.serverID = serverID

	mux := http.NewServeMux()
	mux.HandleFunc("/inventory/report", e.handleReport)

	e.httpServer = &http.Server{
		Addr:         ":9878",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	listener, err := net.Listen("tcp", ":9878")
	if err != nil {
		return fmt.Errorf("listen :9878: %w", err)
	}

	go func() {
		if err := e.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			e.logger.Error("inventory callback server error", zap.Error(err))
		}
	}()

	return nil
}

// handleReport receives hardware inventory from PXE-booted server.
func (e *PXEInventoryExecutor) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
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

	// Determine server_id: from query param, report body, or stored state
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		if sid, ok := report["server_id"].(string); ok {
			serverID = sid
		}
	}
	if serverID == "" {
		e.mu.Lock()
		serverID = e.serverID
		e.mu.Unlock()
	}

	e.logger.Info("received PXE inventory report",
		zap.String("server_id", serverID))

	// Forward to panel via WebSocket event
	e.wsClient.SendEvent("inventory.result", map[string]interface{}{
		"server_id": serverID,
		"data":      report,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (e *PXEInventoryExecutor) cleanup(confPath, pxeCfgPath string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		e.httpServer.Shutdown(ctx)
		e.httpServer = nil
	}

	os.Remove(confPath)
	os.Remove(pxeCfgPath)
	exec.Command("systemctl", "reload", "dnsmasq").Run()
}
