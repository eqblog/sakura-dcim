package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
}

func NewUserService(userRepo repository.UserRepository, roleRepo repository.RoleRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

func (s *UserService) Create(ctx context.Context, tenantID uuid.UUID, req *domain.UserCreateRequest) (*domain.User, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, tenantID, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already exists")
	}

	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Email:        req.Email,
		PasswordHash: hash,
		Name:         req.Name,
		RoleID:       req.RoleID,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if user.RoleID != nil {
		role, err := s.roleRepo.GetByID(ctx, *user.RoleID)
		if err == nil {
			user.Role = role
		}
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.User], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.userRepo.List(ctx, tenantID, page, pageSize)
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req *domain.UserUpdateRequest) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.RoleID != nil {
		user.RoleID = req.RoleID
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := crypto.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		user.PasswordHash = hash
		if err := s.userRepo.UpdatePassword(ctx, id, hash); err != nil {
			return nil, fmt.Errorf("update password: %w", err)
		}
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	if user.RoleID != nil {
		role, err := s.roleRepo.GetByID(ctx, *user.RoleID)
		if err == nil {
			user.Role = role
		}
	}

	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.Delete(ctx, id)
}
