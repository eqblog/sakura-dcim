package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error)
	GetByEmailAnyTenant(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.User], error)
	Update(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*domain.Tenant, error)
	List(ctx context.Context, parentID *uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Tenant], error)
	ListChildren(ctx context.Context, parentID uuid.UUID) ([]domain.Tenant, error)
	GetSubTree(ctx context.Context, rootID uuid.UUID) ([]domain.Tenant, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type RoleRepository interface {
	Create(ctx context.Context, role *domain.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	List(ctx context.Context, tenantID *uuid.UUID) ([]domain.Role, error)
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ServerRepository interface {
	Create(ctx context.Context, server *domain.Server) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Server, error)
	List(ctx context.Context, params domain.ServerListParams) (*domain.PaginatedResult[domain.Server], error)
	Update(ctx context.Context, server *domain.Server) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ServerStatus) error
}

type AgentRepository interface {
	Create(ctx context.Context, agent *domain.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	List(ctx context.Context, page, pageSize int) (*domain.PaginatedResult[domain.Agent], error)
	Update(ctx context.Context, agent *domain.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AgentStatus) error
	UpdateLastSeen(ctx context.Context, id uuid.UUID) error
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, params domain.AuditLogListParams) (*domain.PaginatedResult[domain.AuditLog], error)
}

type OSProfileRepository interface {
	Create(ctx context.Context, profile *domain.OSProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.OSProfile, error)
	List(ctx context.Context, activeOnly bool) ([]domain.OSProfile, error)
	Update(ctx context.Context, profile *domain.OSProfile) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type DiskLayoutRepository interface {
	Create(ctx context.Context, layout *domain.DiskLayout) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.DiskLayout, error)
	List(ctx context.Context) ([]domain.DiskLayout, error)
	Update(ctx context.Context, layout *domain.DiskLayout) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ScriptRepository interface {
	Create(ctx context.Context, script *domain.Script) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Script, error)
	List(ctx context.Context) ([]domain.Script, error)
	ListByOSProfileID(ctx context.Context, osProfileID uuid.UUID) ([]domain.Script, error)
	Update(ctx context.Context, script *domain.Script) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type InstallTaskRepository interface {
	Create(ctx context.Context, task *domain.InstallTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.InstallTask, error)
	GetActiveByServerID(ctx context.Context, serverID uuid.UUID) (*domain.InstallTask, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InstallTaskStatus, progress int, log string) error
}

type InventoryRepository interface {
	Upsert(ctx context.Context, inv *domain.ServerInventory) error
	ListByServerID(ctx context.Context, serverID uuid.UUID) ([]domain.ServerInventory, error)
	DeleteByServerID(ctx context.Context, serverID uuid.UUID) error
}

type IPPoolRepository interface {
	Create(ctx context.Context, pool *domain.IPPool) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.IPPool, error)
	List(ctx context.Context, tenantID *uuid.UUID) ([]domain.IPPool, error)
	Update(ctx context.Context, pool *domain.IPPool) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type IPAddressRepository interface {
	Create(ctx context.Context, addr *domain.IPAddress) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.IPAddress, error)
	ListByPoolID(ctx context.Context, poolID uuid.UUID) ([]domain.IPAddress, error)
	ListByServerID(ctx context.Context, serverID uuid.UUID) ([]domain.IPAddress, error)
	Update(ctx context.Context, addr *domain.IPAddress) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetNextAvailable(ctx context.Context, poolID uuid.UUID) (*domain.IPAddress, error)
}

type SwitchRepository interface {
	Create(ctx context.Context, sw *domain.Switch) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Switch, error)
	List(ctx context.Context) ([]domain.Switch, error)
	Update(ctx context.Context, sw *domain.Switch) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type SwitchPortRepository interface {
	Create(ctx context.Context, port *domain.SwitchPort) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SwitchPort, error)
	ListBySwitchID(ctx context.Context, switchID uuid.UUID) ([]domain.SwitchPort, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]domain.SwitchPort, error)
	Update(ctx context.Context, port *domain.SwitchPort) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpsertBySwitchAndIndex(ctx context.Context, port *domain.SwitchPort) error
}

type DiscoverySessionRepository interface {
	Create(ctx context.Context, session *domain.DiscoverySession) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.DiscoverySession, error)
	GetActiveByAgentID(ctx context.Context, agentID uuid.UUID) (*domain.DiscoverySession, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DiscoverySessionStatus) error
}

type DiscoveredServerRepository interface {
	Upsert(ctx context.Context, ds *domain.DiscoveredServer) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.DiscoveredServer, error)
	List(ctx context.Context, params domain.DiscoveredServerListParams) (*domain.PaginatedResult[domain.DiscoveredServer], error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DiscoveredServerStatus) error
	SetServerID(ctx context.Context, id uuid.UUID, serverID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}
