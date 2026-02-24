package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type IPService struct {
	poolRepo repository.IPPoolRepository
	addrRepo repository.IPAddressRepository
}

func NewIPService(poolRepo repository.IPPoolRepository, addrRepo repository.IPAddressRepository) *IPService {
	return &IPService{poolRepo: poolRepo, addrRepo: addrRepo}
}

// Pool CRUD

func (s *IPService) ListPools(ctx context.Context, tenantID *uuid.UUID) ([]domain.IPPool, error) {
	return s.poolRepo.List(ctx, tenantID)
}

func (s *IPService) GetPool(ctx context.Context, id uuid.UUID) (*domain.IPPool, error) {
	return s.poolRepo.GetByID(ctx, id)
}

func (s *IPService) CreatePool(ctx context.Context, pool *domain.IPPool) (*domain.IPPool, error) {
	if err := s.poolRepo.Create(ctx, pool); err != nil {
		return nil, err
	}
	return s.poolRepo.GetByID(ctx, pool.ID)
}

func (s *IPService) UpdatePool(ctx context.Context, id uuid.UUID, pool *domain.IPPool) (*domain.IPPool, error) {
	pool.ID = id
	if err := s.poolRepo.Update(ctx, pool); err != nil {
		return nil, err
	}
	return s.poolRepo.GetByID(ctx, id)
}

func (s *IPService) DeletePool(ctx context.Context, id uuid.UUID) error {
	return s.poolRepo.Delete(ctx, id)
}

// Address CRUD

func (s *IPService) ListAddresses(ctx context.Context, poolID uuid.UUID) ([]domain.IPAddress, error) {
	return s.addrRepo.ListByPoolID(ctx, poolID)
}

func (s *IPService) ListAddressesByServer(ctx context.Context, serverID uuid.UUID) ([]domain.IPAddress, error) {
	return s.addrRepo.ListByServerID(ctx, serverID)
}

func (s *IPService) CreateAddress(ctx context.Context, poolID uuid.UUID, addr *domain.IPAddress) (*domain.IPAddress, error) {
	addr.PoolID = poolID
	if addr.Status == "" {
		addr.Status = "available"
	}
	if err := s.addrRepo.Create(ctx, addr); err != nil {
		return nil, err
	}
	return s.addrRepo.GetByID(ctx, addr.ID)
}

func (s *IPService) UpdateAddress(ctx context.Context, id uuid.UUID, req *domain.IPAddressUpdateRequest) (*domain.IPAddress, error) {
	addr, err := s.addrRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("address not found: %w", err)
	}
	if req.Status != nil {
		addr.Status = *req.Status
	}
	if req.Note != nil {
		addr.Note = *req.Note
	}
	addr.ServerID = req.ServerID
	if err := s.addrRepo.Update(ctx, addr); err != nil {
		return nil, err
	}
	return s.addrRepo.GetByID(ctx, id)
}

func (s *IPService) DeleteAddress(ctx context.Context, id uuid.UUID) error {
	return s.addrRepo.Delete(ctx, id)
}

// AssignNextAvailable assigns the next available IP from a pool to a server.
func (s *IPService) AssignNextAvailable(ctx context.Context, poolID uuid.UUID, serverID uuid.UUID) (*domain.IPAddress, error) {
	addr, err := s.addrRepo.GetNextAvailable(ctx, poolID)
	if err != nil {
		return nil, err
	}
	addr.ServerID = &serverID
	addr.Status = "assigned"
	if err := s.addrRepo.Update(ctx, addr); err != nil {
		return nil, err
	}
	return addr, nil
}
