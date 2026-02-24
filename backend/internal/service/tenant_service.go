package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type TenantService struct {
	tenantRepo repository.TenantRepository
}

func NewTenantService(tenantRepo repository.TenantRepository) *TenantService {
	return &TenantService{tenantRepo: tenantRepo}
}

type TenantCreateRequest struct {
	Name         string     `json:"name" binding:"required"`
	Slug         string     `json:"slug" binding:"required"`
	ParentID     *uuid.UUID `json:"parent_id"`
	CustomDomain *string    `json:"custom_domain"`
	LogoURL      *string    `json:"logo_url"`
	PrimaryColor *string    `json:"primary_color"`
	FaviconURL   *string    `json:"favicon_url"`
}

type TenantUpdateRequest struct {
	Name         *string `json:"name"`
	Slug         *string `json:"slug"`
	CustomDomain *string `json:"custom_domain"`
	LogoURL      *string `json:"logo_url"`
	PrimaryColor *string `json:"primary_color"`
	FaviconURL   *string `json:"favicon_url"`
}

func (s *TenantService) Create(ctx context.Context, req *TenantCreateRequest) (*domain.Tenant, error) {
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	existing, _ := s.tenantRepo.GetBySlug(ctx, slug)
	if existing != nil {
		return nil, fmt.Errorf("slug already exists")
	}

	tenant := &domain.Tenant{
		ID:           uuid.New(),
		ParentID:     req.ParentID,
		Name:         req.Name,
		Slug:         slug,
		CustomDomain: req.CustomDomain,
		LogoURL:      req.LogoURL,
		PrimaryColor: req.PrimaryColor,
		FaviconURL:   req.FaviconURL,
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	return tenant, nil
}

func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

func (s *TenantService) List(ctx context.Context, parentID *uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Tenant], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.tenantRepo.List(ctx, parentID, page, pageSize)
}

func (s *TenantService) Update(ctx context.Context, id uuid.UUID, req *TenantUpdateRequest) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*req.Slug))
		existing, _ := s.tenantRepo.GetBySlug(ctx, slug)
		if existing != nil && existing.ID != id {
			return nil, fmt.Errorf("slug already exists")
		}
		tenant.Slug = slug
	}
	if req.CustomDomain != nil {
		tenant.CustomDomain = req.CustomDomain
	}
	if req.LogoURL != nil {
		tenant.LogoURL = req.LogoURL
	}
	if req.PrimaryColor != nil {
		tenant.PrimaryColor = req.PrimaryColor
	}
	if req.FaviconURL != nil {
		tenant.FaviconURL = req.FaviconURL
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	return tenant, nil
}

func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.tenantRepo.Delete(ctx, id)
}
