package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type ServerService struct {
	serverRepo repository.ServerRepository
	cfg        *config.Config
}

func NewServerService(serverRepo repository.ServerRepository, cfg *config.Config) *ServerService {
	return &ServerService{
		serverRepo: serverRepo,
		cfg:        cfg,
	}
}

func (s *ServerService) Create(ctx context.Context, tenantID uuid.UUID, req *domain.ServerCreateRequest) (*domain.Server, error) {
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	server := &domain.Server{
		ID:        uuid.New(),
		TenantID:  &tenantID,
		AgentID:   req.AgentID,
		Hostname:  req.Hostname,
		Label:     req.Label,
		Status:    domain.ServerStatusActive,
		PrimaryIP: req.PrimaryIP,
		Tags:      tags,
		Notes:     req.Notes,
	}

	if req.IPMIIP != "" {
		server.IPMIIP = req.IPMIIP
	}
	if req.IPMIUser != "" {
		encrypted, err := crypto.EncryptAESGCM(req.IPMIUser, s.cfg.Crypto.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt ipmi_user: %w", err)
		}
		server.IPMIUser = encrypted
	}
	if req.IPMIPass != "" {
		encrypted, err := crypto.EncryptAESGCM(req.IPMIPass, s.cfg.Crypto.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt ipmi_pass: %w", err)
		}
		server.IPMIPass = encrypted
	}

	if err := s.serverRepo.Create(ctx, server); err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}

	server.IPMIUser = ""
	server.IPMIPass = ""
	return server, nil
}

func (s *ServerService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error) {
	server, err := s.serverRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	server.IPMIUser = ""
	server.IPMIPass = ""
	return server, nil
}

func (s *ServerService) List(ctx context.Context, params domain.ServerListParams) (*domain.PaginatedResult[domain.Server], error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	result, err := s.serverRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	for i := range result.Items {
		result.Items[i].IPMIUser = ""
		result.Items[i].IPMIPass = ""
	}
	return result, nil
}

func (s *ServerService) Update(ctx context.Context, id uuid.UUID, req *domain.ServerUpdateRequest) (*domain.Server, error) {
	server, err := s.serverRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	if req.Hostname != nil {
		server.Hostname = *req.Hostname
	}
	if req.Label != nil {
		server.Label = *req.Label
	}
	if req.AgentID != nil {
		server.AgentID = req.AgentID
	}
	if req.PrimaryIP != nil {
		server.PrimaryIP = *req.PrimaryIP
	}
	if req.IPMIIP != nil {
		server.IPMIIP = *req.IPMIIP
	}
	if req.IPMIUser != nil && *req.IPMIUser != "" {
		encrypted, err := crypto.EncryptAESGCM(*req.IPMIUser, s.cfg.Crypto.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt ipmi_user: %w", err)
		}
		server.IPMIUser = encrypted
	}
	if req.IPMIPass != nil && *req.IPMIPass != "" {
		encrypted, err := crypto.EncryptAESGCM(*req.IPMIPass, s.cfg.Crypto.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt ipmi_pass: %w", err)
		}
		server.IPMIPass = encrypted
	}
	if req.Tags != nil {
		server.Tags = *req.Tags
	}
	if req.Notes != nil {
		server.Notes = *req.Notes
	}

	if err := s.serverRepo.Update(ctx, server); err != nil {
		return nil, fmt.Errorf("update server: %w", err)
	}

	server.IPMIUser = ""
	server.IPMIPass = ""
	return server, nil
}

func (s *ServerService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.serverRepo.Delete(ctx, id)
}

func (s *ServerService) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ServerStatus) error {
	return s.serverRepo.UpdateStatus(ctx, id, status)
}
