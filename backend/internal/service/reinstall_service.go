package service

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

type ReinstallService struct {
	serverRepo      repository.ServerRepository
	osProfileRepo   repository.OSProfileRepository
	diskLayoutRepo  repository.DiskLayoutRepository
	scriptRepo      repository.ScriptRepository
	installTaskRepo repository.InstallTaskRepository
	hub             *ws.Hub
	cfg             *config.Config
	logger          *zap.Logger
}

func NewReinstallService(
	serverRepo repository.ServerRepository,
	osProfileRepo repository.OSProfileRepository,
	diskLayoutRepo repository.DiskLayoutRepository,
	scriptRepo repository.ScriptRepository,
	installTaskRepo repository.InstallTaskRepository,
	hub *ws.Hub,
	cfg *config.Config,
	logger *zap.Logger,
) *ReinstallService {
	return &ReinstallService{
		serverRepo:      serverRepo,
		osProfileRepo:   osProfileRepo,
		diskLayoutRepo:  diskLayoutRepo,
		scriptRepo:      scriptRepo,
		installTaskRepo: installTaskRepo,
		hub:             hub,
		cfg:             cfg,
		logger:          logger,
	}
}

// AutoRAIDLevel decides the RAID level based on disk count.
func AutoRAIDLevel(diskCount int) string {
	switch {
	case diskCount <= 1:
		return "none"
	case diskCount == 2:
		return "raid1"
	case diskCount == 3:
		return "raid5"
	default:
		return "raid10"
	}
}

// StartReinstall initiates an OS reinstallation for a server.
// netCfg is optional — when provided, gateway/netmask/DNS are injected into the template and PXE payload.
func (s *ReinstallService) StartReinstall(ctx context.Context, serverID uuid.UUID, req *domain.ReinstallRequest, netCfg *domain.NetworkConfig) (*domain.InstallTask, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	if server.AgentID == nil {
		return nil, fmt.Errorf("server has no agent assigned")
	}

	if !s.hub.IsAgentOnline(*server.AgentID) {
		return nil, fmt.Errorf("agent is offline")
	}

	// Check for active install task
	existing, _ := s.installTaskRepo.GetActiveByServerID(ctx, serverID)
	if existing != nil {
		return nil, fmt.Errorf("server already has an active install task (status: %s)", existing.Status)
	}

	// Validate OS profile
	osProfile, err := s.osProfileRepo.GetByID(ctx, req.OSProfileID)
	if err != nil {
		return nil, fmt.Errorf("OS profile not found: %w", err)
	}

	if !osProfile.IsActive {
		return nil, fmt.Errorf("OS profile is not active")
	}

	// Handle RAID level
	raidLevel := req.RAIDLevel
	if raidLevel == "" || raidLevel == "auto" {
		raidLevel = "auto"
	}

	// Hash root password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.RootPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Create install task
	sshKeys := req.SSHKeys
	if sshKeys == nil {
		sshKeys = []string{}
	}
	task := &domain.InstallTask{
		ServerID:         serverID,
		OSProfileID:      req.OSProfileID,
		DiskLayoutID:     req.DiskLayoutID,
		RAIDLevel:        raidLevel,
		Status:           domain.InstallStatusPending,
		RootPasswordHash: string(passwordHash),
		SSHKeys:          sshKeys,
		Progress:         0,
		Log:              "",
	}

	if err := s.installTaskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create install task: %w", err)
	}

	// Update server status
	_ = s.serverRepo.UpdateStatus(ctx, serverID, domain.ServerStatusReinstalling)

	// Render preseed/kickstart template
	renderedTemplate, err := s.renderTemplate(osProfile, server, req, netCfg)
	if err != nil {
		s.logger.Warn("failed to render template", zap.Error(err))
	}

	// Decrypt IPMI credentials for PXE boot trigger
	ipmiUser, ipmiPass := "", ""
	if server.IPMIUser != "" {
		ipmiUser, _ = crypto.DecryptAESGCM(server.IPMIUser, s.cfg.Crypto.EncryptionKey)
	}
	if server.IPMIPass != "" {
		ipmiPass, _ = crypto.DecryptAESGCM(server.IPMIPass, s.cfg.Crypto.EncryptionKey)
	}

	// Send PXE prepare command to agent
	pxePayload := map[string]interface{}{
		"task_id":      task.ID.String(),
		"server_id":    serverID.String(),
		"server_mac":   server.MACAddress,
		"server_ip":    server.PrimaryIP,
		"kernel_url":   osProfile.KernelURL,
		"initrd_url":   osProfile.InitrdURL,
		"boot_args":    osProfile.BootArgs,
		"template":     renderedTemplate,
		"raid_level":   raidLevel,
		"ipmi_ip":      server.IPMIIP,
		"ipmi_user":    ipmiUser,
		"ipmi_pass":    ipmiPass,
		"ssh_keys":     req.SSHKeys,
		"gateway":      "",
		"netmask":      "",
		"nameservers":  []string{},
	}
	if netCfg != nil {
		pxePayload["gateway"] = netCfg.Gateway
		pxePayload["netmask"] = netCfg.Netmask
		pxePayload["nameservers"] = netCfg.Nameservers
	}

	go func() {
		// Update task to PXE booting
		_ = s.installTaskRepo.UpdateStatus(context.Background(), task.ID, domain.InstallStatusPXEBooting, 5, "Sending PXE prepare to agent...\n")

		_, err := s.hub.SendRequest(*server.AgentID, ws.ActionPXEPrepare, pxePayload, 60*time.Second)
		if err != nil {
			s.logger.Error("PXE prepare failed", zap.Error(err))
			_ = s.installTaskRepo.UpdateStatus(context.Background(), task.ID, domain.InstallStatusFailed, 0, fmt.Sprintf("PXE prepare failed: %s\n", err))
			_ = s.serverRepo.UpdateStatus(context.Background(), serverID, domain.ServerStatusError)
			return
		}

		_ = s.installTaskRepo.UpdateStatus(context.Background(), task.ID, domain.InstallStatusPXEBooting, 10, "PXE environment prepared. Waiting for server to boot...\n")
	}()

	s.logger.Info("reinstall started",
		zap.String("server_id", serverID.String()),
		zap.String("task_id", task.ID.String()),
		zap.String("os_profile", osProfile.Name),
	)

	return task, nil
}

// GetInstallStatus returns the current install task for a server.
func (s *ReinstallService) GetInstallStatus(ctx context.Context, serverID uuid.UUID) (*domain.InstallTask, error) {
	return s.installTaskRepo.GetActiveByServerID(ctx, serverID)
}

// GetInstallTask returns a specific install task by ID.
func (s *ReinstallService) GetInstallTask(ctx context.Context, id uuid.UUID) (*domain.InstallTask, error) {
	return s.installTaskRepo.GetByID(ctx, id)
}

// HandlePXEStatusEvent processes PXE installation progress events from agents.
func (s *ReinstallService) HandlePXEStatusEvent(agentID uuid.UUID, msg *ws.Message) {
	var payload ws.PXEStatusPayload
	// Parse payload from the message
	raw, ok := msg.Payload.(map[string]interface{})
	if !ok {
		s.logger.Warn("invalid pxe.status payload")
		return
	}

	if v, ok := raw["server_id"].(string); ok {
		payload.ServerID = v
	}
	if v, ok := raw["status"].(string); ok {
		payload.Status = v
	}
	if v, ok := raw["progress"].(float64); ok {
		payload.Progress = int(v)
	}
	if v, ok := raw["message"].(string); ok {
		payload.Message = v
	}

	serverID, err := uuid.Parse(payload.ServerID)
	if err != nil {
		s.logger.Warn("invalid server_id in pxe.status", zap.String("server_id", payload.ServerID))
		return
	}

	task, err := s.installTaskRepo.GetActiveByServerID(context.Background(), serverID)
	if err != nil {
		s.logger.Warn("no active task for pxe.status", zap.Error(err))
		return
	}

	var status domain.InstallTaskStatus
	switch payload.Status {
	case "pxe_booting":
		status = domain.InstallStatusPXEBooting
	case "installing":
		status = domain.InstallStatusInstalling
	case "post_scripts":
		status = domain.InstallStatusPostScripts
	case "completed":
		status = domain.InstallStatusCompleted
	case "failed":
		status = domain.InstallStatusFailed
	default:
		status = task.Status
	}

	logMsg := ""
	if payload.Message != "" {
		logMsg = payload.Message + "\n"
	}

	_ = s.installTaskRepo.UpdateStatus(context.Background(), task.ID, status, payload.Progress, logMsg)

	// Update server status on completion/failure
	if status == domain.InstallStatusCompleted {
		_ = s.serverRepo.UpdateStatus(context.Background(), serverID, domain.ServerStatusActive)
		go s.cleanupPXE(serverID)
	} else if status == domain.InstallStatusFailed {
		_ = s.serverRepo.UpdateStatus(context.Background(), serverID, domain.ServerStatusError)
	}

	s.logger.Info("install progress",
		zap.String("server_id", serverID.String()),
		zap.String("status", string(status)),
		zap.Int("progress", payload.Progress),
	)
}

// cleanupPXE sends a PXE cleanup command to the agent after install completion.
func (s *ReinstallService) cleanupPXE(serverID uuid.UUID) {
	server, err := s.serverRepo.GetByID(context.Background(), serverID)
	if err != nil || server.AgentID == nil {
		return
	}
	payload := map[string]interface{}{
		"server_id":  serverID.String(),
		"server_mac": server.MACAddress,
	}
	_, err = s.hub.SendRequest(*server.AgentID, ws.ActionPXECleanup, payload, 30*time.Second)
	if err != nil {
		s.logger.Warn("PXE cleanup failed", zap.Error(err), zap.String("server_id", serverID.String()))
	}
}

// renderTemplate renders a Kickstart/Preseed/cloud-init template with server variables.
func (s *ReinstallService) renderTemplate(profile *domain.OSProfile, server *domain.Server, req *domain.ReinstallRequest, netCfg *domain.NetworkConfig) (string, error) {
	if profile.Template == "" {
		return "", nil
	}

	tmpl, err := template.New("install").Parse(profile.Template)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	data := map[string]interface{}{
		"Hostname":     server.Hostname,
		"PrimaryIP":    server.PrimaryIP,
		"RootPassword": req.RootPassword,
		"SSHKeys":      req.SSHKeys,
		"RAIDLevel":    req.RAIDLevel,
		"MACAddress":   server.MACAddress,
		"Gateway":      "",
		"Netmask":      "",
		"Nameservers":  []string{},
	}
	if netCfg != nil {
		data["Gateway"] = netCfg.Gateway
		data["Netmask"] = netCfg.Netmask
		data["Nameservers"] = netCfg.Nameservers
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
