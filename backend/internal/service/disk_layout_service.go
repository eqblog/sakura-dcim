package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type DiskLayoutService struct {
	repo repository.DiskLayoutRepository
}

func NewDiskLayoutService(repo repository.DiskLayoutRepository) *DiskLayoutService {
	return &DiskLayoutService{repo: repo}
}

func (s *DiskLayoutService) Create(ctx context.Context, layout *domain.DiskLayout) (*domain.DiskLayout, error) {
	if layout.Tags == nil {
		layout.Tags = []string{}
	}
	if err := s.repo.Create(ctx, layout); err != nil {
		return nil, err
	}
	return layout, nil
}

func (s *DiskLayoutService) GetByID(ctx context.Context, id uuid.UUID) (*domain.DiskLayout, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DiskLayoutService) List(ctx context.Context) ([]domain.DiskLayout, error) {
	return s.repo.List(ctx)
}

func (s *DiskLayoutService) Update(ctx context.Context, id uuid.UUID, layout *domain.DiskLayout) (*domain.DiskLayout, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Name = layout.Name
	existing.Description = layout.Description
	existing.Layout = layout.Layout
	existing.Tags = layout.Tags
	if existing.Tags == nil {
		existing.Tags = []string{}
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *DiskLayoutService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
