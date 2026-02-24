package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"go.uber.org/zap"
)

// PXEExecutor manages PXE boot environment (dnsmasq DHCP/TFTP).
type PXEExecutor struct {
	logger  *zap.Logger
	tftpDir string // TFTP root directory
	confDir string // dnsmasq config directory
}

func NewPXEExecutor(logger *zap.Logger) *PXEExecutor {
	return &PXEExecutor{
		logger:  logger,
		tftpDir: "/srv/tftp",
		confDir: "/etc/dnsmasq.d",
	}
}

type PXEPreparePayload struct {
	TaskID    string   `json:"task_id"`
	ServerID  string   `json:"server_id"`
	ServerMAC string   `json:"server_mac"`
	ServerIP  string   `json:"server_ip"`
	KernelURL string   `json:"kernel_url"`
	InitrdURL string   `json:"initrd_url"`
	BootArgs  string   `json:"boot_args"`
	Template  string   `json:"template"`
	RAIDLevel string   `json:"raid_level"`
	IPMIIP    string   `json:"ipmi_ip"`
	IPMIUser  string   `json:"ipmi_user"`
	IPMIPass  string   `json:"ipmi_pass"`
	SSHKeys   []string `json:"ssh_keys"`
}

type PXECleanupPayload struct {
	ServerID  string `json:"server_id"`
	ServerMAC string `json:"server_mac"`
}

// HandlePXEPrepare sets up PXE boot environment for a server.
func (e *PXEExecutor) HandlePXEPrepare(raw json.RawMessage) (interface{}, error) {
	var p PXEPreparePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("preparing PXE boot",
		zap.String("server_id", p.ServerID),
		zap.String("server_ip", p.ServerIP),
	)

	// Create server-specific TFTP directory
	serverDir := filepath.Join(e.tftpDir, p.ServerID)
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return nil, fmt.Errorf("create tftp dir: %w", err)
	}

	// Download kernel and initrd if URLs provided
	if p.KernelURL != "" {
		if err := e.downloadFile(p.KernelURL, filepath.Join(serverDir, "vmlinuz")); err != nil {
			return nil, fmt.Errorf("download kernel: %w", err)
		}
	}
	if p.InitrdURL != "" {
		if err := e.downloadFile(p.InitrdURL, filepath.Join(serverDir, "initrd.img")); err != nil {
			return nil, fmt.Errorf("download initrd: %w", err)
		}
	}

	// Write preseed/kickstart template
	if p.Template != "" {
		templatePath := filepath.Join(serverDir, "preseed.cfg")
		if err := os.WriteFile(templatePath, []byte(p.Template), 0644); err != nil {
			return nil, fmt.Errorf("write template: %w", err)
		}
	}

	// Generate PXE boot config
	pxeConfig, err := e.generatePXEConfig(&p)
	if err != nil {
		return nil, fmt.Errorf("generate pxe config: %w", err)
	}

	// Write PXE config for this server
	pxeConfigPath := filepath.Join(e.tftpDir, "pxelinux.cfg", e.macToFilename(p.ServerMAC))
	if err := os.MkdirAll(filepath.Dir(pxeConfigPath), 0755); err != nil {
		return nil, fmt.Errorf("create pxelinux.cfg dir: %w", err)
	}
	if err := os.WriteFile(pxeConfigPath, []byte(pxeConfig), 0644); err != nil {
		return nil, fmt.Errorf("write pxe config: %w", err)
	}

	// Write dnsmasq host config for DHCP reservation
	if p.ServerMAC != "" && p.ServerIP != "" {
		dnsmasqConf := fmt.Sprintf("dhcp-host=%s,%s,set:pxe-%s\n", p.ServerMAC, p.ServerIP, p.ServerID)
		confPath := filepath.Join(e.confDir, fmt.Sprintf("pxe-%s.conf", p.ServerID))
		if err := os.WriteFile(confPath, []byte(dnsmasqConf), 0644); err != nil {
			e.logger.Warn("failed to write dnsmasq config", zap.Error(err))
		}
	}

	// Reload dnsmasq to pick up new config
	e.reloadDnsmasq()

	// Set PXE boot via IPMI if credentials provided
	if p.IPMIIP != "" && p.IPMIUser != "" {
		e.setPXEBoot(p.IPMIIP, p.IPMIUser, p.IPMIPass)
	}

	return map[string]string{
		"status":  "prepared",
		"task_id": p.TaskID,
	}, nil
}

// HandlePXECleanup removes PXE boot config for a server.
func (e *PXEExecutor) HandlePXECleanup(raw json.RawMessage) (interface{}, error) {
	var p PXECleanupPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	e.logger.Info("cleaning up PXE", zap.String("server_id", p.ServerID))

	// Remove server TFTP directory
	serverDir := filepath.Join(e.tftpDir, p.ServerID)
	os.RemoveAll(serverDir)

	// Remove PXE config
	if p.ServerMAC != "" {
		pxeConfigPath := filepath.Join(e.tftpDir, "pxelinux.cfg", e.macToFilename(p.ServerMAC))
		os.Remove(pxeConfigPath)
	}

	// Remove dnsmasq config
	confPath := filepath.Join(e.confDir, fmt.Sprintf("pxe-%s.conf", p.ServerID))
	os.Remove(confPath)

	e.reloadDnsmasq()

	return map[string]string{"status": "cleaned"}, nil
}

func (e *PXEExecutor) downloadFile(url, dest string) error {
	cmd := exec.Command("curl", "-sSL", "-o", dest, url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("curl: %s: %w", string(out), err)
	}
	return nil
}

func (e *PXEExecutor) generatePXEConfig(p *PXEPreparePayload) (string, error) {
	tmplStr := `DEFAULT install
LABEL install
    KERNEL {{.ServerID}}/vmlinuz
    APPEND initrd={{.ServerID}}/initrd.img {{.BootArgs}}
    IPAPPEND 2
`
	tmpl, err := template.New("pxe").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, p); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *PXEExecutor) macToFilename(mac string) string {
	if mac == "" {
		return "default"
	}
	// PXE config filename format: 01-xx-xx-xx-xx-xx-xx
	mac = strings.ReplaceAll(mac, ":", "-")
	return "01-" + strings.ToLower(mac)
}

func (e *PXEExecutor) reloadDnsmasq() {
	cmd := exec.Command("systemctl", "reload", "dnsmasq")
	if err := cmd.Run(); err != nil {
		e.logger.Debug("dnsmasq reload failed (may not be installed)", zap.Error(err))
	}
}

func (e *PXEExecutor) setPXEBoot(ip, user, pass string) {
	// Set next boot to PXE
	cmd := exec.Command("ipmitool", "-I", "lanplus", "-H", ip, "-U", user, "-P", pass,
		"chassis", "bootdev", "pxe")
	if out, err := cmd.CombinedOutput(); err != nil {
		e.logger.Warn("failed to set PXE boot via IPMI", zap.Error(err), zap.String("output", string(out)))
		return
	}

	// Power reset to trigger PXE boot
	cmd = exec.Command("ipmitool", "-I", "lanplus", "-H", ip, "-U", user, "-P", pass,
		"chassis", "power", "reset")
	if out, err := cmd.CombinedOutput(); err != nil {
		e.logger.Warn("failed to reset server for PXE boot", zap.Error(err), zap.String("output", string(out)))
	}
}
