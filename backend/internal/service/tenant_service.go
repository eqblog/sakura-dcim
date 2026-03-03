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
	KvmMode      *string    `json:"kvm_mode"`
}

type TenantUpdateRequest struct {
	Name         *string `json:"name"`
	Slug         *string `json:"slug"`
	CustomDomain *string `json:"custom_domain"`
	LogoURL      *string `json:"logo_url"`
	PrimaryColor *string `json:"primary_color"`
	FaviconURL   *string `json:"favicon_url"`
	KvmMode      *string `json:"kvm_mode"`
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
		KvmMode:      "webkvm",
	}
	if req.KvmMode != nil && (*req.KvmMode == "webkvm" || *req.KvmMode == "vconsole") {
		tenant.KvmMode = *req.KvmMode
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
	if req.KvmMode != nil && (*req.KvmMode == "webkvm" || *req.KvmMode == "vconsole") {
		tenant.KvmMode = *req.KvmMode
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	return tenant, nil
}

func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.tenantRepo.Delete(ctx, id)
}

// ListChildren returns direct child tenants of a parent (reseller hierarchy).
func (s *TenantService) ListChildren(ctx context.Context, parentID uuid.UUID) ([]domain.Tenant, error) {
	return s.tenantRepo.ListChildren(ctx, parentID)
}

// GetSubTree returns the entire tenant tree rooted at the given tenant (recursive CTE).
func (s *TenantService) GetSubTree(ctx context.Context, rootID uuid.UUID) ([]domain.Tenant, error) {
	return s.tenantRepo.GetSubTree(ctx, rootID)
}

// TenantTree is a nested structure for reseller hierarchy display.
type TenantTree struct {
	domain.Tenant
	Children []*TenantTree `json:"children,omitempty"`
}

// BuildTree converts a flat tenant list into a nested tree rooted at rootID.
func BuildTree(flat []domain.Tenant, rootID uuid.UUID) *TenantTree {
	byID := make(map[uuid.UUID]*TenantTree)
	for _, t := range flat {
		byID[t.ID] = &TenantTree{Tenant: t}
	}

	// Build parent→children links using pointers (order-independent)
	for _, node := range byID {
		if node.ParentID != nil {
			if parent, ok := byID[*node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	return byID[rootID]
}

// GetHierarchy builds a nested tree from a flat list of tenants.
func (s *TenantService) GetHierarchy(ctx context.Context, rootID uuid.UUID) (*TenantTree, error) {
	flat, err := s.tenantRepo.GetSubTree(ctx, rootID)
	if err != nil {
		return nil, fmt.Errorf("get sub tree: %w", err)
	}
	if len(flat) == 0 {
		return nil, fmt.Errorf("tenant not found")
	}

	root := BuildTree(flat, rootID)
	if root == nil {
		return nil, fmt.Errorf("root tenant not found in tree")
	}
	return root, nil
}
