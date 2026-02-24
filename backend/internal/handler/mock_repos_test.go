package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

// ── pagination helper ────────────────────────────────────────────────

func paginate[T any](items []T, page, pageSize int) *domain.PaginatedResult[T] {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return &domain.PaginatedResult[T]{
		Items:      items[start:end],
		Total:      int64(total),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// ── memUserRepo ──────────────────────────────────────────────────────

type memUserRepo struct {
	mu    sync.Mutex
	users map[uuid.UUID]*domain.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{users: make(map[uuid.UUID]*domain.User)}
}

func (r *memUserRepo) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	u.CreatedAt = time.Now()
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *memUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *u
	return &cp, nil
}

func (r *memUserRepo) GetByEmail(_ context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.TenantID == tenantID && u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (r *memUserRepo) GetByEmailAnyTenant(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (r *memUserRepo) List(_ context.Context, tenantID uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.User], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.User
	for _, u := range r.users {
		if u.TenantID == tenantID {
			items = append(items, *u)
		}
	}
	return paginate(items, page, pageSize), nil
}

func (r *memUserRepo) Update(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *memUserRepo) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	u.PasswordHash = hash
	return nil
}

func (r *memUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.users, id)
	return nil
}

func (r *memUserRepo) UpdateLastLogin(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	now := time.Now()
	u.LastLogin = &now
	return nil
}

// ── memTenantRepo ────────────────────────────────────────────────────

type memTenantRepo struct {
	mu      sync.Mutex
	tenants map[uuid.UUID]*domain.Tenant
}

func newMemTenantRepo() *memTenantRepo {
	return &memTenantRepo{tenants: make(map[uuid.UUID]*domain.Tenant)}
}

func (r *memTenantRepo) Create(_ context.Context, t *domain.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	cp := *t
	r.tenants[t.ID] = &cp
	return nil
}

func (r *memTenantRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tenants[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *t
	return &cp, nil
}

func (r *memTenantRepo) GetBySlug(_ context.Context, slug string) (*domain.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tenants {
		if t.Slug == slug {
			cp := *t
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (r *memTenantRepo) GetByDomain(_ context.Context, d string) (*domain.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tenants {
		if t.CustomDomain != nil && *t.CustomDomain == d {
			cp := *t
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (r *memTenantRepo) List(_ context.Context, parentID *uuid.UUID, page, pageSize int) (*domain.PaginatedResult[domain.Tenant], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.Tenant
	for _, t := range r.tenants {
		if parentID != nil {
			if t.ParentID != nil && *t.ParentID == *parentID {
				items = append(items, *t)
			}
		} else {
			items = append(items, *t)
		}
	}
	return paginate(items, page, pageSize), nil
}

func (r *memTenantRepo) ListChildren(_ context.Context, parentID uuid.UUID) ([]domain.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.Tenant
	for _, t := range r.tenants {
		if t.ParentID != nil && *t.ParentID == parentID {
			items = append(items, *t)
		}
	}
	return items, nil
}

func (r *memTenantRepo) GetSubTree(_ context.Context, rootID uuid.UUID) ([]domain.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []domain.Tenant
	var walk func(id uuid.UUID)
	walk = func(id uuid.UUID) {
		t, ok := r.tenants[id]
		if !ok {
			return
		}
		result = append(result, *t)
		for _, child := range r.tenants {
			if child.ParentID != nil && *child.ParentID == id {
				walk(child.ID)
			}
		}
	}
	walk(rootID)
	return result, nil
}

func (r *memTenantRepo) Update(_ context.Context, t *domain.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t.UpdatedAt = time.Now()
	cp := *t
	r.tenants[t.ID] = &cp
	return nil
}

func (r *memTenantRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tenants, id)
	return nil
}

// ── memRoleRepo ──────────────────────────────────────────────────────

type memRoleRepo struct {
	mu    sync.Mutex
	roles map[uuid.UUID]*domain.Role
}

func newMemRoleRepo() *memRoleRepo {
	return &memRoleRepo{roles: make(map[uuid.UUID]*domain.Role)}
}

func (r *memRoleRepo) Create(_ context.Context, role *domain.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	role.CreatedAt = time.Now()
	cp := *role
	r.roles[role.ID] = &cp
	return nil
}

func (r *memRoleRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	role, ok := r.roles[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *role
	return &cp, nil
}

func (r *memRoleRepo) List(_ context.Context, tenantID *uuid.UUID) ([]domain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.Role
	for _, role := range r.roles {
		if tenantID == nil || (role.TenantID != nil && *role.TenantID == *tenantID) {
			items = append(items, *role)
		}
	}
	return items, nil
}

func (r *memRoleRepo) Update(_ context.Context, role *domain.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *role
	r.roles[role.ID] = &cp
	return nil
}

func (r *memRoleRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.roles, id)
	return nil
}

// ── memServerRepo ────────────────────────────────────────────────────

type memServerRepo struct {
	mu      sync.Mutex
	servers map[uuid.UUID]*domain.Server
}

func newMemServerRepo() *memServerRepo {
	return &memServerRepo{servers: make(map[uuid.UUID]*domain.Server)}
}

func (r *memServerRepo) Create(_ context.Context, s *domain.Server) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	cp := *s
	r.servers[s.ID] = &cp
	return nil
}

func (r *memServerRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Server, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.servers[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *s
	return &cp, nil
}

func (r *memServerRepo) List(_ context.Context, params domain.ServerListParams) (*domain.PaginatedResult[domain.Server], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.Server
	for _, s := range r.servers {
		if params.TenantID != nil && (s.TenantID == nil || *s.TenantID != *params.TenantID) {
			continue
		}
		if params.Status != nil && s.Status != *params.Status {
			continue
		}
		if params.AgentID != nil && (s.AgentID == nil || *s.AgentID != *params.AgentID) {
			continue
		}
		if params.Search != "" {
			search := strings.ToLower(params.Search)
			if !strings.Contains(strings.ToLower(s.Hostname), search) &&
				!strings.Contains(strings.ToLower(s.Label), search) &&
				!strings.Contains(strings.ToLower(s.PrimaryIP), search) {
				continue
			}
		}
		items = append(items, *s)
	}
	return paginate(items, params.Page, params.PageSize), nil
}

func (r *memServerRepo) Update(_ context.Context, s *domain.Server) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s.UpdatedAt = time.Now()
	cp := *s
	r.servers[s.ID] = &cp
	return nil
}

func (r *memServerRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.servers, id)
	return nil
}

func (r *memServerRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.ServerStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.servers[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	s.Status = status
	s.UpdatedAt = time.Now()
	return nil
}

// ── memAgentRepo ─────────────────────────────────────────────────────

type memAgentRepo struct {
	mu     sync.Mutex
	agents map[uuid.UUID]*domain.Agent
}

func newMemAgentRepo() *memAgentRepo {
	return &memAgentRepo{agents: make(map[uuid.UUID]*domain.Agent)}
}

func (r *memAgentRepo) Create(_ context.Context, a *domain.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.CreatedAt = time.Now()
	cp := *a
	r.agents[a.ID] = &cp
	return nil
}

func (r *memAgentRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.agents[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *a
	return &cp, nil
}

func (r *memAgentRepo) List(_ context.Context, page, pageSize int) (*domain.PaginatedResult[domain.Agent], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.Agent
	for _, a := range r.agents {
		items = append(items, *a)
	}
	return paginate(items, page, pageSize), nil
}

func (r *memAgentRepo) Update(_ context.Context, a *domain.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *a
	r.agents[a.ID] = &cp
	return nil
}

func (r *memAgentRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, id)
	return nil
}

func (r *memAgentRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.AgentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.agents[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	a.Status = status
	return nil
}

func (r *memAgentRepo) UpdateLastSeen(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.agents[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	now := time.Now()
	a.LastSeen = &now
	return nil
}

// ── memAuditLogRepo ──────────────────────────────────────────────────

type memAuditLogRepo struct {
	mu   sync.Mutex
	logs []domain.AuditLog
}

func newMemAuditLogRepo() *memAuditLogRepo {
	return &memAuditLogRepo{}
}

func (r *memAuditLogRepo) Create(_ context.Context, l *domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	l.CreatedAt = time.Now()
	r.logs = append(r.logs, *l)
	return nil
}

func (r *memAuditLogRepo) List(_ context.Context, params domain.AuditLogListParams) (*domain.PaginatedResult[domain.AuditLog], error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.AuditLog
	for _, l := range r.logs {
		if params.Action != "" && !strings.Contains(l.Action, params.Action) {
			continue
		}
		if params.ResourceType != "" && l.ResourceType != params.ResourceType {
			continue
		}
		items = append(items, l)
	}
	return paginate(items, params.Page, params.PageSize), nil
}

// ── memOSProfileRepo ─────────────────────────────────────────────────

type memOSProfileRepo struct {
	mu       sync.Mutex
	profiles map[uuid.UUID]*domain.OSProfile
}

func newMemOSProfileRepo() *memOSProfileRepo {
	return &memOSProfileRepo{profiles: make(map[uuid.UUID]*domain.OSProfile)}
}

func (r *memOSProfileRepo) Create(_ context.Context, p *domain.OSProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.CreatedAt = time.Now()
	cp := *p
	r.profiles[p.ID] = &cp
	return nil
}

func (r *memOSProfileRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.OSProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *p
	return &cp, nil
}

func (r *memOSProfileRepo) List(_ context.Context, activeOnly bool) ([]domain.OSProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.OSProfile
	for _, p := range r.profiles {
		if activeOnly && !p.IsActive {
			continue
		}
		items = append(items, *p)
	}
	return items, nil
}

func (r *memOSProfileRepo) Update(_ context.Context, p *domain.OSProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.profiles[p.ID] = &cp
	return nil
}

func (r *memOSProfileRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.profiles, id)
	return nil
}

// ── memDiskLayoutRepo ────────────────────────────────────────────────

type memDiskLayoutRepo struct {
	mu      sync.Mutex
	layouts map[uuid.UUID]*domain.DiskLayout
}

func newMemDiskLayoutRepo() *memDiskLayoutRepo {
	return &memDiskLayoutRepo{layouts: make(map[uuid.UUID]*domain.DiskLayout)}
}

func (r *memDiskLayoutRepo) Create(_ context.Context, l *domain.DiskLayout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	l.CreatedAt = time.Now()
	cp := *l
	r.layouts[l.ID] = &cp
	return nil
}

func (r *memDiskLayoutRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.DiskLayout, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.layouts[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *l
	return &cp, nil
}

func (r *memDiskLayoutRepo) List(_ context.Context) ([]domain.DiskLayout, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.DiskLayout
	for _, l := range r.layouts {
		items = append(items, *l)
	}
	return items, nil
}

func (r *memDiskLayoutRepo) Update(_ context.Context, l *domain.DiskLayout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *l
	r.layouts[l.ID] = &cp
	return nil
}

func (r *memDiskLayoutRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.layouts, id)
	return nil
}

// ── memScriptRepo ────────────────────────────────────────────────────

type memScriptRepo struct {
	mu      sync.Mutex
	scripts map[uuid.UUID]*domain.Script
}

func newMemScriptRepo() *memScriptRepo {
	return &memScriptRepo{scripts: make(map[uuid.UUID]*domain.Script)}
}

func (r *memScriptRepo) Create(_ context.Context, s *domain.Script) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.CreatedAt = time.Now()
	cp := *s
	r.scripts[s.ID] = &cp
	return nil
}

func (r *memScriptRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Script, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.scripts[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *s
	return &cp, nil
}

func (r *memScriptRepo) List(_ context.Context) ([]domain.Script, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.Script
	for _, s := range r.scripts {
		items = append(items, *s)
	}
	return items, nil
}

func (r *memScriptRepo) ListByOSProfileID(_ context.Context, osProfileID uuid.UUID) ([]domain.Script, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pid := osProfileID.String()
	var items []domain.Script
	for _, s := range r.scripts {
		for _, id := range s.OSProfileIDs {
			if id == pid {
				items = append(items, *s)
				break
			}
		}
	}
	return items, nil
}

func (r *memScriptRepo) Update(_ context.Context, s *domain.Script) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	r.scripts[s.ID] = &cp
	return nil
}

func (r *memScriptRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.scripts, id)
	return nil
}

// ── memInstallTaskRepo ───────────────────────────────────────────────

type memInstallTaskRepo struct {
	mu    sync.Mutex
	tasks map[uuid.UUID]*domain.InstallTask
}

func newMemInstallTaskRepo() *memInstallTaskRepo {
	return &memInstallTaskRepo{tasks: make(map[uuid.UUID]*domain.InstallTask)}
}

func (r *memInstallTaskRepo) Create(_ context.Context, t *domain.InstallTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	t.CreatedAt = time.Now()
	cp := *t
	r.tasks[t.ID] = &cp
	return nil
}

func (r *memInstallTaskRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.InstallTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *t
	return &cp, nil
}

func (r *memInstallTaskRepo) GetActiveByServerID(_ context.Context, serverID uuid.UUID) (*domain.InstallTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tasks {
		if t.ServerID == serverID && t.Status != domain.InstallStatusCompleted && t.Status != domain.InstallStatusFailed {
			cp := *t
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (r *memInstallTaskRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.InstallTaskStatus, progress int, log string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	t.Status = status
	t.Progress = progress
	if log != "" {
		t.Log += log + "\n"
	}
	return nil
}

// ── memInventoryRepo ─────────────────────────────────────────────────

type memInventoryRepo struct {
	mu    sync.Mutex
	items map[string]*domain.ServerInventory // key: serverID:component
}

func newMemInventoryRepo() *memInventoryRepo {
	return &memInventoryRepo{items: make(map[string]*domain.ServerInventory)}
}

func invKey(serverID uuid.UUID, component string) string {
	return serverID.String() + ":" + component
}

func (r *memInventoryRepo) Upsert(_ context.Context, inv *domain.ServerInventory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	inv.CollectedAt = time.Now()
	cp := *inv
	r.items[invKey(inv.ServerID, inv.Component)] = &cp
	return nil
}

func (r *memInventoryRepo) ListByServerID(_ context.Context, serverID uuid.UUID) ([]domain.ServerInventory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	prefix := serverID.String() + ":"
	var items []domain.ServerInventory
	for k, v := range r.items {
		if strings.HasPrefix(k, prefix) {
			items = append(items, *v)
		}
	}
	return items, nil
}

func (r *memInventoryRepo) DeleteByServerID(_ context.Context, serverID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	prefix := serverID.String() + ":"
	for k := range r.items {
		if strings.HasPrefix(k, prefix) {
			delete(r.items, k)
		}
	}
	return nil
}

// ── memIPPoolRepo ────────────────────────────────────────────────────

type memIPPoolRepo struct {
	mu    sync.Mutex
	pools map[uuid.UUID]*domain.IPPool
}

func newMemIPPoolRepo() *memIPPoolRepo {
	return &memIPPoolRepo{pools: make(map[uuid.UUID]*domain.IPPool)}
}

func (r *memIPPoolRepo) Create(_ context.Context, p *domain.IPPool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	cp := *p
	r.pools[p.ID] = &cp
	return nil
}

func (r *memIPPoolRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.IPPool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.pools[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *p
	return &cp, nil
}

func (r *memIPPoolRepo) List(_ context.Context, tenantID *uuid.UUID) ([]domain.IPPool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.IPPool
	for _, p := range r.pools {
		items = append(items, *p)
	}
	return items, nil
}

func (r *memIPPoolRepo) Update(_ context.Context, p *domain.IPPool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.pools[p.ID] = &cp
	return nil
}

func (r *memIPPoolRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pools, id)
	return nil
}

// ── memIPAddressRepo ─────────────────────────────────────────────────

type memIPAddressRepo struct {
	mu    sync.Mutex
	addrs map[uuid.UUID]*domain.IPAddress
}

func newMemIPAddressRepo() *memIPAddressRepo {
	return &memIPAddressRepo{addrs: make(map[uuid.UUID]*domain.IPAddress)}
}

func (r *memIPAddressRepo) Create(_ context.Context, a *domain.IPAddress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	cp := *a
	r.addrs[a.ID] = &cp
	return nil
}

func (r *memIPAddressRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.IPAddress, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.addrs[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *a
	return &cp, nil
}

func (r *memIPAddressRepo) ListByPoolID(_ context.Context, poolID uuid.UUID) ([]domain.IPAddress, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.IPAddress
	for _, a := range r.addrs {
		if a.PoolID == poolID {
			items = append(items, *a)
		}
	}
	return items, nil
}

func (r *memIPAddressRepo) ListByServerID(_ context.Context, serverID uuid.UUID) ([]domain.IPAddress, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.IPAddress
	for _, a := range r.addrs {
		if a.ServerID != nil && *a.ServerID == serverID {
			items = append(items, *a)
		}
	}
	return items, nil
}

func (r *memIPAddressRepo) Update(_ context.Context, a *domain.IPAddress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *a
	r.addrs[a.ID] = &cp
	return nil
}

func (r *memIPAddressRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.addrs, id)
	return nil
}

func (r *memIPAddressRepo) GetNextAvailable(_ context.Context, poolID uuid.UUID) (*domain.IPAddress, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.addrs {
		if a.PoolID == poolID && a.Status == "available" {
			cp := *a
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("no available addresses")
}

// ── memSwitchRepo ────────────────────────────────────────────────────

type memSwitchRepo struct {
	mu       sync.Mutex
	switches map[uuid.UUID]*domain.Switch
}

func newMemSwitchRepo() *memSwitchRepo {
	return &memSwitchRepo{switches: make(map[uuid.UUID]*domain.Switch)}
}

func (r *memSwitchRepo) Create(_ context.Context, sw *domain.Switch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sw.ID == uuid.Nil {
		sw.ID = uuid.New()
	}
	sw.CreatedAt = time.Now()
	sw.UpdatedAt = time.Now()
	cp := *sw
	r.switches[sw.ID] = &cp
	return nil
}

func (r *memSwitchRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Switch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sw, ok := r.switches[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *sw
	return &cp, nil
}

func (r *memSwitchRepo) List(_ context.Context) ([]domain.Switch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.Switch
	for _, sw := range r.switches {
		items = append(items, *sw)
	}
	return items, nil
}

func (r *memSwitchRepo) Update(_ context.Context, sw *domain.Switch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sw.UpdatedAt = time.Now()
	cp := *sw
	r.switches[sw.ID] = &cp
	return nil
}

func (r *memSwitchRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.switches, id)
	return nil
}

// ── memSwitchPortRepo ────────────────────────────────────────────────

type memSwitchPortRepo struct {
	mu    sync.Mutex
	ports map[uuid.UUID]*domain.SwitchPort
}

func newMemSwitchPortRepo() *memSwitchPortRepo {
	return &memSwitchPortRepo{ports: make(map[uuid.UUID]*domain.SwitchPort)}
}

func (r *memSwitchPortRepo) Create(_ context.Context, p *domain.SwitchPort) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	cp := *p
	r.ports[p.ID] = &cp
	return nil
}

func (r *memSwitchPortRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.SwitchPort, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.ports[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *p
	return &cp, nil
}

func (r *memSwitchPortRepo) ListBySwitchID(_ context.Context, switchID uuid.UUID) ([]domain.SwitchPort, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.SwitchPort
	for _, p := range r.ports {
		if p.SwitchID == switchID {
			items = append(items, *p)
		}
	}
	return items, nil
}

func (r *memSwitchPortRepo) GetByServerID(_ context.Context, serverID uuid.UUID) ([]domain.SwitchPort, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []domain.SwitchPort
	for _, p := range r.ports {
		if p.ServerID != nil && *p.ServerID == serverID {
			items = append(items, *p)
		}
	}
	return items, nil
}

func (r *memSwitchPortRepo) Update(_ context.Context, p *domain.SwitchPort) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.ports[p.ID] = &cp
	return nil
}

func (r *memSwitchPortRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.ports, id)
	return nil
}
