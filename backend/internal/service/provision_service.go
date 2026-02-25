package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

// ProvisionService orchestrates the full server provisioning flow:
// validate → assign IP → switch VLAN push → PXE install.
type ProvisionService struct {
	serverRepo   repository.ServerRepository
	ipSvc        *IPService
	reinstallSvc *ReinstallService
	portRepo     repository.SwitchPortRepository
	hub          *ws.Hub
	logger       *zap.Logger
}

func NewProvisionService(
	serverRepo repository.ServerRepository,
	ipSvc *IPService,
	reinstallSvc *ReinstallService,
	portRepo repository.SwitchPortRepository,
	hub *ws.Hub,
	logger *zap.Logger,
) *ProvisionService {
	return &ProvisionService{
		serverRepo:   serverRepo,
		ipSvc:        ipSvc,
		reinstallSvc: reinstallSvc,
		portRepo:     portRepo,
		hub:          hub,
		logger:       logger,
	}
}

// PreflightResult contains pre-flight validation results.
type PreflightResult struct {
	HasMAC        bool     `json:"has_mac"`
	HasAgent      bool     `json:"has_agent"`
	AgentOnline   bool     `json:"agent_online"`
	HasIP         bool     `json:"has_ip"`
	HasSwitchPort bool     `json:"has_switch_port"`
	Warnings      []string `json:"warnings"`
}

// Preflight checks whether a server is ready for provisioning.
func (s *ProvisionService) Preflight(ctx context.Context, serverID uuid.UUID) (*PreflightResult, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	result := &PreflightResult{
		Warnings: []string{},
	}

	result.HasMAC = server.MACAddress != ""
	result.HasAgent = server.AgentID != nil
	if server.AgentID != nil {
		result.AgentOnline = s.hub.IsAgentOnline(*server.AgentID)
	}

	addrs, _ := s.ipSvc.ListAddressesByServer(ctx, serverID)
	result.HasIP = len(addrs) > 0

	if s.portRepo != nil {
		ports, _ := s.portRepo.GetByServerID(ctx, serverID)
		result.HasSwitchPort = len(ports) > 0
	}

	if !result.HasMAC {
		result.Warnings = append(result.Warnings, "Server has no MAC address — PXE DHCP reservation will use fallback config")
	}
	if !result.AgentOnline {
		result.Warnings = append(result.Warnings, "Agent is offline — provisioning will fail")
	}
	if !result.HasSwitchPort {
		result.Warnings = append(result.Warnings, "No switch port linked — VLAN automation will be skipped")
	}

	return result, nil
}

// Provision executes the full provisioning flow.
func (s *ProvisionService) Provision(ctx context.Context, serverID uuid.UUID, req *domain.ProvisionRequest) (*domain.InstallTask, error) {
	// 1. Load and validate server
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
	if server.MACAddress == "" {
		s.logger.Warn("provisioning server without MAC address", zap.String("server_id", serverID.String()))
	}

	// 2. Set server status to provisioning
	if err := s.serverRepo.UpdateStatus(ctx, serverID, domain.ServerStatusProvisioning); err != nil {
		return nil, fmt.Errorf("update server status: %w", err)
	}

	// 3. Assign IP if server doesn't already have one
	addrs, _ := s.ipSvc.ListAddressesByServer(ctx, serverID)
	if len(addrs) == 0 {
		assignResult, err := s.ipSvc.AutoAssign(ctx, serverID, req.PoolID, req.VRF, domain.VLANActionExecute)
		if err != nil {
			_ = s.serverRepo.UpdateStatus(ctx, serverID, domain.ServerStatusError)
			return nil, fmt.Errorf("IP assignment failed: %w", err)
		}

		// Update server's primary_ip field
		server.PrimaryIP = assignResult.Address.Address
		if err := s.serverRepo.Update(ctx, server); err != nil {
			s.logger.Warn("failed to update server primary_ip", zap.Error(err))
		}

		s.logger.Info("IP assigned during provisioning",
			zap.String("server_id", serverID.String()),
			zap.String("ip", assignResult.Address.Address),
		)
	}

	// 4. Resolve network config from pool
	netCfg, err := s.ipSvc.GetNetworkConfigForServer(ctx, serverID)
	if err != nil {
		s.logger.Warn("could not resolve network config",
			zap.Error(err), zap.String("server_id", serverID.String()))
		netCfg = &domain.NetworkConfig{}
	}

	// 5. Delegate to ReinstallService for PXE install
	reinstallReq := &domain.ReinstallRequest{
		OSProfileID:  req.OSProfileID,
		DiskLayoutID: req.DiskLayoutID,
		RAIDLevel:    req.RAIDLevel,
		RootPassword: req.RootPassword,
		SSHKeys:      req.SSHKeys,
	}

	task, err := s.reinstallSvc.StartReinstall(ctx, serverID, reinstallReq, netCfg)
	if err != nil {
		_ = s.serverRepo.UpdateStatus(ctx, serverID, domain.ServerStatusError)
		return nil, fmt.Errorf("PXE install failed to start: %w", err)
	}

	s.logger.Info("provisioning started",
		zap.String("server_id", serverID.String()),
		zap.String("task_id", task.ID.String()),
	)

	return task, nil
}
