package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type ScriptService struct {
	repo repository.ScriptRepository
}

func NewScriptService(repo repository.ScriptRepository) *ScriptService {
	return &ScriptService{repo: repo}
}

func (s *ScriptService) Create(ctx context.Context, script *domain.Script) (*domain.Script, error) {
	if err := s.repo.Create(ctx, script); err != nil {
		return nil, err
	}
	return script, nil
}

func (s *ScriptService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Script, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ScriptService) List(ctx context.Context) ([]domain.Script, error) {
	return s.repo.List(ctx)
}

func (s *ScriptService) ListByOSProfileID(ctx context.Context, osProfileID uuid.UUID) ([]domain.Script, error) {
	return s.repo.ListByOSProfileID(ctx, osProfileID)
}

func (s *ScriptService) Update(ctx context.Context, id uuid.UUID, script *domain.Script) (*domain.Script, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Name = script.Name
	existing.Description = script.Description
	existing.Content = script.Content
	existing.RunOrder = script.RunOrder
	existing.OSProfileIDs = script.OSProfileIDs
	existing.Tags = script.Tags

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *ScriptService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
