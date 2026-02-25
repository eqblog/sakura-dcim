package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/google/uuid"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository"
)

type IPService struct {
	poolRepo  repository.IPPoolRepository
	addrRepo  repository.IPAddressRepository
	switchSvc *SwitchService
	portRepo  repository.SwitchPortRepository
}

func NewIPService(poolRepo repository.IPPoolRepository, addrRepo repository.IPAddressRepository) *IPService {
	return &IPService{poolRepo: poolRepo, addrRepo: addrRepo}
}

// SetSwitchDeps injects switch service dependencies for port auto-provisioning.
func (s *IPService) SetSwitchDeps(switchSvc *SwitchService, portRepo repository.SwitchPortRepository) {
	s.switchSvc = switchSvc
	s.portRepo = portRepo
}

// Pool CRUD

func (s *IPService) ListPools(ctx context.Context, tenantID *uuid.UUID) ([]domain.IPPool, error) {
	return s.poolRepo.List(ctx, tenantID)
}

func (s *IPService) ListChildPools(ctx context.Context, parentID uuid.UUID) ([]domain.IPPool, error) {
	return s.poolRepo.ListByParentID(ctx, parentID)
}

func (s *IPService) GetPool(ctx context.Context, id uuid.UUID) (*domain.IPPool, error) {
	return s.poolRepo.GetByID(ctx, id)
}

func (s *IPService) CreatePool(ctx context.Context, pool *domain.IPPool) (*domain.IPPool, error) {
	if pool.PoolType == "" {
		pool.PoolType = "ip_pool"
	}

	// Validate child CIDR is within parent
	if pool.ParentID != nil {
		parent, err := s.poolRepo.GetByID(ctx, *pool.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parent pool not found: %w", err)
		}
		if !cidrContains(parent.Network, pool.Network) {
			return nil, fmt.Errorf("child CIDR %s is not within parent CIDR %s", pool.Network, parent.Network)
		}
	}

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

// GeneratePoolIPs generates all host IP addresses for a pool's CIDR range.
func (s *IPService) GeneratePoolIPs(ctx context.Context, poolID uuid.UUID, reserveGateway bool) error {
	pool, err := s.poolRepo.GetByID(ctx, poolID)
	if err != nil {
		return fmt.Errorf("pool not found: %w", err)
	}

	_, ipNet, err := net.ParseCIDR(pool.Network)
	if err != nil {
		return fmt.Errorf("invalid CIDR %s: %w", pool.Network, err)
	}

	gatewayIP := net.ParseIP(pool.Gateway)

	ones, bits := ipNet.Mask.Size()
	if bits-ones > 20 {
		return fmt.Errorf("CIDR /%d too large (max /%d for IP generation)", ones, bits-20)
	}

	// Enumerate host IPs (skip network and broadcast)
	networkIP := ipNet.IP.To4()
	if networkIP == nil {
		return fmt.Errorf("only IPv4 is supported for IP generation")
	}

	netInt := binary.BigEndian.Uint32(networkIP)
	maskInt := binary.BigEndian.Uint32(net.IP(ipNet.Mask).To4())
	broadcast := netInt | ^maskInt

	for ip := netInt + 1; ip < broadcast; ip++ {
		ipBytes := make(net.IP, 4)
		binary.BigEndian.PutUint32(ipBytes, ip)
		addrStr := ipBytes.String()

		status := "available"
		if reserveGateway && gatewayIP != nil && ipBytes.Equal(gatewayIP) {
			status = "reserved"
		}

		addr := &domain.IPAddress{
			PoolID:  poolID,
			Address: addrStr,
			Status:  status,
		}
		if err := s.addrRepo.Create(ctx, addr); err != nil {
			// Skip duplicates (address already exists)
			continue
		}
	}

	return nil
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

	oldStatus := addr.Status
	oldServerID := addr.ServerID

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

	// Switch automation: unprovision when IP is unassigned from a server
	if oldStatus == "assigned" && addr.Status != "assigned" && oldServerID != nil {
		s.unprovisionServerPort(ctx, addr.PoolID, *oldServerID)
	}
	// Switch automation: provision when IP is newly assigned to a server
	if addr.Status == "assigned" && addr.ServerID != nil && (oldStatus != "assigned" || oldServerID == nil) {
		s.provisionServerPort(ctx, addr.PoolID, *addr.ServerID)
	}

	return s.addrRepo.GetByID(ctx, id)
}

func (s *IPService) DeleteAddress(ctx context.Context, id uuid.UUID) error {
	// Unprovision switch port if the address was assigned
	addr, err := s.addrRepo.GetByID(ctx, id)
	if err == nil && addr.Status == "assigned" && addr.ServerID != nil {
		s.unprovisionServerPort(ctx, addr.PoolID, *addr.ServerID)
	}
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

	// Switch automation: provision server's switch port with pool's VLAN config
	s.provisionServerPort(ctx, poolID, serverID)

	return addr, nil
}

// provisionServerPort configures the server's linked switch port based on the pool's VLAN settings.
func (s *IPService) provisionServerPort(ctx context.Context, poolID uuid.UUID, serverID uuid.UUID) {
	if s.switchSvc == nil || s.portRepo == nil {
		return
	}
	pool, err := s.poolRepo.GetByID(ctx, poolID)
	if err != nil || !pool.SwitchAutomation {
		return
	}
	ports, err := s.portRepo.GetByServerID(ctx, serverID)
	if err != nil || len(ports) == 0 {
		return
	}
	for _, port := range ports {
		port.PortMode = pool.VlanMode
		switch pool.VlanMode {
		case "access":
			port.VlanID = pool.VlanID
			port.NativeVlanID = 0
			port.TrunkVlans = ""
		case "trunk_native":
			port.NativeVlanID = pool.NativeVlanID
			port.TrunkVlans = pool.TrunkVlans
			port.VlanID = 0
		case "trunk":
			port.TrunkVlans = pool.TrunkVlans
			port.VlanID = 0
			port.NativeVlanID = 0
		}
		_ = s.portRepo.Update(ctx, &port)
		_ = s.switchSvc.ProvisionPort(ctx, port.SwitchID, port.ID)
	}
}

// unprovisionServerPort reverts the server's linked switch port to default access mode.
func (s *IPService) unprovisionServerPort(ctx context.Context, poolID uuid.UUID, serverID uuid.UUID) {
	if s.switchSvc == nil || s.portRepo == nil {
		return
	}
	pool, err := s.poolRepo.GetByID(ctx, poolID)
	if err != nil || !pool.SwitchAutomation {
		return
	}
	ports, err := s.portRepo.GetByServerID(ctx, serverID)
	if err != nil || len(ports) == 0 {
		return
	}
	for _, port := range ports {
		port.PortMode = "access"
		port.VlanID = 1
		port.NativeVlanID = 0
		port.TrunkVlans = ""
		_ = s.portRepo.Update(ctx, &port)
		_ = s.switchSvc.ProvisionPort(ctx, port.SwitchID, port.ID)
	}
}

// cidrContains checks if childCIDR is fully within parentCIDR.
func cidrContains(parentCIDR, childCIDR string) bool {
	_, parentNet, err := net.ParseCIDR(parentCIDR)
	if err != nil {
		return false
	}
	childIP, childNet, err := net.ParseCIDR(childCIDR)
	if err != nil {
		return false
	}

	// Child network address must be within parent, and child prefix must be longer (or equal)
	pOnes, _ := parentNet.Mask.Size()
	cOnes, _ := childNet.Mask.Size()
	return parentNet.Contains(childIP) && cOnes >= pOnes
}
