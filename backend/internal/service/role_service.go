package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type RoleService struct {
	roleRepo repository.RoleRepository
}

func NewRoleService(roleRepo repository.RoleRepository) *RoleService {
	return &RoleService{roleRepo: roleRepo}
}

func (s *RoleService) Create(ctx context.Context, tenantID *uuid.UUID, name string, permissions []string) (*domain.Role, error) {
	role := &domain.Role{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Permissions: permissions,
		IsSystem:    false,
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	return role, nil
}

func (s *RoleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *RoleService) List(ctx context.Context, tenantID *uuid.UUID) ([]domain.Role, error) {
	return s.roleRepo.List(ctx, tenantID)
}

func (s *RoleService) Update(ctx context.Context, id uuid.UUID, name string, permissions []string) (*domain.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("role not found: %w", err)
	}

	if role.IsSystem {
		return nil, fmt.Errorf("cannot modify system role")
	}

	role.Name = name
	role.Permissions = permissions

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}

	return role, nil
}

func (s *RoleService) Delete(ctx context.Context, id uuid.UUID) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	if role.IsSystem {
		return fmt.Errorf("cannot delete system role")
	}

	return s.roleRepo.Delete(ctx, id)
}

func (s *RoleService) AllPermissions() []string {
	return domain.AllPermissions
}
