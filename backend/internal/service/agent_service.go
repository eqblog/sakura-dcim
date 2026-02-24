package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type AgentService struct {
	agentRepo repository.AgentRepository
}

func NewAgentService(agentRepo repository.AgentRepository) *AgentService {
	return &AgentService{agentRepo: agentRepo}
}

func (s *AgentService) Create(ctx context.Context, req *domain.AgentCreateRequest) (*domain.AgentCreateResponse, error) {
	token, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	tokenHash, err := crypto.HashPassword(token)
	if err != nil {
		return nil, fmt.Errorf("hash token: %w", err)
	}

	agent := &domain.Agent{
		ID:           uuid.New(),
		Name:         req.Name,
		Location:     req.Location,
		TokenHash:    tokenHash,
		Status:       domain.AgentStatusOffline,
		Capabilities: req.Capabilities,
	}

	if err := s.agentRepo.Create(ctx, agent); err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return &domain.AgentCreateResponse{
		Agent: agent,
		Token: token,
	}, nil
}

func (s *AgentService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	return s.agentRepo.GetByID(ctx, id)
}

func (s *AgentService) List(ctx context.Context, page, pageSize int) (*domain.PaginatedResult[domain.Agent], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.agentRepo.List(ctx, page, pageSize)
}

func (s *AgentService) Update(ctx context.Context, id uuid.UUID, name, location string, capabilities []string) (*domain.Agent, error) {
	agent, err := s.agentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	agent.Name = name
	agent.Location = location
	if capabilities != nil {
		agent.Capabilities = capabilities
	}

	if err := s.agentRepo.Update(ctx, agent); err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}

	return agent, nil
}

func (s *AgentService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.agentRepo.Delete(ctx, id)
}

func (s *AgentService) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	return s.agentRepo.UpdateLastSeen(ctx, id)
}

func (s *AgentService) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AgentStatus) error {
	return s.agentRepo.UpdateStatus(ctx, id, status)
}
