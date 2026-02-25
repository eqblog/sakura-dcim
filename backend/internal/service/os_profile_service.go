package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type OSProfileService struct {
	repo repository.OSProfileRepository
}

func NewOSProfileService(repo repository.OSProfileRepository) *OSProfileService {
	return &OSProfileService{repo: repo}
}

func (s *OSProfileService) Create(ctx context.Context, profile *domain.OSProfile) (*domain.OSProfile, error) {
	if profile.Tags == nil {
		profile.Tags = []string{}
	}
	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *OSProfileService) GetByID(ctx context.Context, id uuid.UUID) (*domain.OSProfile, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OSProfileService) List(ctx context.Context, activeOnly bool) ([]domain.OSProfile, error) {
	return s.repo.List(ctx, activeOnly)
}

func (s *OSProfileService) Update(ctx context.Context, id uuid.UUID, profile *domain.OSProfile) (*domain.OSProfile, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Name = profile.Name
	existing.OSFamily = profile.OSFamily
	existing.Version = profile.Version
	existing.Arch = profile.Arch
	existing.KernelURL = profile.KernelURL
	existing.InitrdURL = profile.InitrdURL
	existing.BootArgs = profile.BootArgs
	existing.TemplateType = profile.TemplateType
	existing.Template = profile.Template
	existing.IsActive = profile.IsActive
	existing.Tags = profile.Tags
	if existing.Tags == nil {
		existing.Tags = []string{}
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *OSProfileService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
