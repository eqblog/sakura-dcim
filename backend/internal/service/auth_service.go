package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserInactive       = errors.New("user account is inactive")
)

type AuthService struct {
	userRepo   repository.UserRepository
	roleRepo   repository.RoleRepository
	tenantRepo repository.TenantRepository
	cfg        *config.Config
}

func NewAuthService(userRepo repository.UserRepository, roleRepo repository.RoleRepository, tenantRepo repository.TenantRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		roleRepo:   roleRepo,
		tenantRepo: tenantRepo,
		cfg:        cfg,
	}
}

type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *domain.User `json:"user"`
}

func (s *AuthService) Login(ctx context.Context, req *domain.UserLoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.GetByEmailAnyTenant(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if !crypto.CheckPassword(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	roleID := ""
	if user.RoleID != nil {
		roleID = user.RoleID.String()
		role, err := s.roleRepo.GetByID(ctx, *user.RoleID)
		if err == nil {
			user.Role = role
		}
	}

	accessToken, err := crypto.GenerateAccessToken(
		user.ID.String(),
		user.TenantID.String(),
		roleID,
		s.cfg.JWT.Secret,
		s.cfg.JWT.AccessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := crypto.GenerateRefreshToken(
		user.ID.String(),
		s.cfg.JWT.Secret,
		s.cfg.JWT.RefreshTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	// Populate tenant branding
	tenant, tErr := s.tenantRepo.GetByID(ctx, user.TenantID)
	if tErr == nil {
		user.Tenant = tenant
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	userIDStr, err := crypto.ParseRefreshToken(refreshToken, s.cfg.JWT.Secret)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user ID in token")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	roleID := ""
	if user.RoleID != nil {
		roleID = user.RoleID.String()
	}

	newAccessToken, err := crypto.GenerateAccessToken(
		user.ID.String(),
		user.TenantID.String(),
		roleID,
		s.cfg.JWT.Secret,
		s.cfg.JWT.AccessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	newRefreshToken, err := crypto.GenerateRefreshToken(
		user.ID.String(),
		s.cfg.JWT.Secret,
		s.cfg.JWT.RefreshTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.RoleID != nil {
		role, err := s.roleRepo.GetByID(ctx, *user.RoleID)
		if err == nil {
			user.Role = role
		}
	}

	// Populate tenant branding info
	tenant, err := s.tenantRepo.GetByID(ctx, user.TenantID)
	if err == nil {
		user.Tenant = tenant
	}

	return user, nil
}

// GetTenantBranding returns public branding info for a tenant (by domain or slug).
func (s *AuthService) GetTenantBranding(ctx context.Context, host string) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetByDomain(ctx, host)
	if err != nil {
		// Fallback: try as slug
		tenant, err = s.tenantRepo.GetBySlug(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("tenant not found for host: %s", host)
		}
	}
	return tenant, nil
}
