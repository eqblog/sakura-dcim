-- Sakura DCIM - Initial Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ===================
-- Tenants (multi-tenant hierarchy)
-- ===================
CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id     UUID REFERENCES tenants(id) ON DELETE SET NULL,
    name          VARCHAR(255) NOT NULL,
    slug          VARCHAR(100) UNIQUE NOT NULL,
    custom_domain VARCHAR(255),
    logo_url      TEXT,
    primary_color VARCHAR(7),
    favicon_url   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_parent ON tenants(parent_id);
CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_domain ON tenants(custom_domain) WHERE custom_domain IS NOT NULL;

-- ===================
-- Roles (RBAC)
-- ===================
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]',
    is_system   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_roles_tenant ON roles(tenant_id);

-- ===================
-- Users
-- ===================
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    name          VARCHAR(255) NOT NULL DEFAULT '',
    role_id       UUID REFERENCES roles(id) ON DELETE SET NULL,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    last_login    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);

-- ===================
-- Agents (remote datacenter agents)
-- ===================
CREATE TABLE agents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(255) NOT NULL,
    location     VARCHAR(255) NOT NULL DEFAULT '',
    token_hash   TEXT NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'offline',
    last_seen    TIMESTAMPTZ,
    version      VARCHAR(50) NOT NULL DEFAULT '',
    capabilities JSONB NOT NULL DEFAULT '[]',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===================
-- Servers
-- ===================
CREATE TABLE servers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id) ON DELETE SET NULL,
    agent_id    UUID REFERENCES agents(id) ON DELETE SET NULL,
    hostname    VARCHAR(255) NOT NULL DEFAULT '',
    label       VARCHAR(255) NOT NULL DEFAULT '',
    status      VARCHAR(30) NOT NULL DEFAULT 'active',
    primary_ip  INET,
    ipmi_ip     INET,
    ipmi_user   TEXT NOT NULL DEFAULT '',
    ipmi_pass   TEXT NOT NULL DEFAULT '',
    cpu_model   VARCHAR(255) NOT NULL DEFAULT '',
    cpu_cores   INT NOT NULL DEFAULT 0,
    ram_mb      BIGINT NOT NULL DEFAULT 0,
    tags        TEXT[] NOT NULL DEFAULT '{}',
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_servers_tenant ON servers(tenant_id);
CREATE INDEX idx_servers_agent ON servers(agent_id);
CREATE INDEX idx_servers_status ON servers(status);
CREATE INDEX idx_servers_tags ON servers USING GIN(tags);

-- ===================
-- Server Inventory
-- ===================
CREATE TABLE server_inventory (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id    UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    component    VARCHAR(50) NOT NULL,
    details      JSONB NOT NULL DEFAULT '{}',
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_server ON server_inventory(server_id);

-- ===================
-- Server Disks
-- ===================
CREATE TABLE server_disks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id  UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    slot       VARCHAR(20) NOT NULL DEFAULT '',
    model      VARCHAR(255) NOT NULL DEFAULT '',
    serial     VARCHAR(255) NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    type       VARCHAR(10) NOT NULL DEFAULT 'hdd',
    health     VARCHAR(20) NOT NULL DEFAULT 'unknown',
    smart_data JSONB
);

CREATE INDEX idx_disks_server ON server_disks(server_id);

-- ===================
-- OS Profiles
-- ===================
CREATE TABLE os_profiles (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    os_family     VARCHAR(50) NOT NULL,
    version       VARCHAR(50) NOT NULL DEFAULT '',
    arch          VARCHAR(10) NOT NULL DEFAULT 'amd64',
    kernel_url    TEXT NOT NULL DEFAULT '',
    initrd_url    TEXT NOT NULL DEFAULT '',
    boot_args     TEXT NOT NULL DEFAULT '',
    template_type VARCHAR(20) NOT NULL DEFAULT 'kickstart',
    template      TEXT NOT NULL DEFAULT '',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    tags          TEXT[] NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===================
-- Disk Layouts
-- ===================
CREATE TABLE disk_layouts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    layout      JSONB NOT NULL DEFAULT '{}',
    tags        TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===================
-- Scripts (post-install)
-- ===================
CREATE TABLE scripts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(255) NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    content        TEXT NOT NULL DEFAULT '',
    run_order      INT NOT NULL DEFAULT 0,
    os_profile_ids UUID[] NOT NULL DEFAULT '{}',
    tags           TEXT[] NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===================
-- Install Tasks
-- ===================
CREATE TABLE install_tasks (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id          UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    os_profile_id      UUID NOT NULL REFERENCES os_profiles(id),
    disk_layout_id     UUID REFERENCES disk_layouts(id),
    raid_level         VARCHAR(10) NOT NULL DEFAULT 'auto',
    status             VARCHAR(30) NOT NULL DEFAULT 'pending',
    root_password_hash TEXT NOT NULL DEFAULT '',
    ssh_keys           TEXT[] NOT NULL DEFAULT '{}',
    progress           INT NOT NULL DEFAULT 0,
    log                TEXT NOT NULL DEFAULT '',
    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_install_tasks_server ON install_tasks(server_id);
CREATE INDEX idx_install_tasks_status ON install_tasks(status);

-- ===================
-- Switches (SNMP)
-- ===================
CREATE TABLE switches (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id       UUID REFERENCES agents(id) ON DELETE SET NULL,
    name           VARCHAR(255) NOT NULL,
    ip             INET NOT NULL,
    snmp_community VARCHAR(255) NOT NULL DEFAULT 'public',
    snmp_version   VARCHAR(5) NOT NULL DEFAULT 'v2c',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ===================
-- Switch Ports
-- ===================
CREATE TABLE switch_ports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    switch_id   UUID NOT NULL REFERENCES switches(id) ON DELETE CASCADE,
    server_id   UUID REFERENCES servers(id) ON DELETE SET NULL,
    port_index  INT NOT NULL,
    port_name   VARCHAR(100) NOT NULL DEFAULT '',
    speed_mbps  INT NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_switch_ports_switch ON switch_ports(switch_id);
CREATE INDEX idx_switch_ports_server ON switch_ports(server_id);

-- ===================
-- IP Pools
-- ===================
CREATE TABLE ip_pools (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id) ON DELETE SET NULL,
    network     CIDR NOT NULL,
    gateway     INET NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_ip_pools_tenant ON ip_pools(tenant_id);

-- ===================
-- IP Addresses
-- ===================
CREATE TABLE ip_addresses (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id   UUID NOT NULL REFERENCES ip_pools(id) ON DELETE CASCADE,
    address   INET NOT NULL UNIQUE,
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    status    VARCHAR(20) NOT NULL DEFAULT 'available',
    note      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_ip_addresses_pool ON ip_addresses(pool_id);
CREATE INDEX idx_ip_addresses_server ON ip_addresses(server_id);
CREATE INDEX idx_ip_addresses_status ON ip_addresses(status);

-- ===================
-- Audit Logs
-- ===================
CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID,
    user_id       UUID,
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL DEFAULT '',
    resource_id   UUID,
    details       JSONB,
    ip_address    INET,
    user_agent    TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant_created ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_user_created ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

-- ===================
-- Seed: Default system tenant, admin role, admin user
-- ===================
INSERT INTO tenants (id, name, slug) VALUES
    ('00000000-0000-0000-0000-000000000001', 'System', 'system');

INSERT INTO roles (id, tenant_id, name, permissions, is_system) VALUES
    ('00000000-0000-0000-0000-000000000001', NULL, 'Super Admin', '["*"]', true),
    ('00000000-0000-0000-0000-000000000002', NULL, 'Admin', '["server.view","server.create","server.edit","server.delete","server.power","ipmi.kvm","ipmi.sensors","os.reinstall","os.profile.manage","raid.manage","bandwidth.view","switch.manage","inventory.view","inventory.scan","user.view","user.manage","role.manage","tenant.view","agent.manage","ip.manage","audit.view","settings.manage","script.manage","disk_layout.manage"]', true),
    ('00000000-0000-0000-0000-000000000003', NULL, 'Customer', '["server.view","server.power","ipmi.kvm","ipmi.sensors","os.reinstall","bandwidth.view","inventory.view"]', true);

-- Default admin user (password: admin123)
-- bcrypt hash of "admin123"
INSERT INTO users (id, tenant_id, email, password_hash, name, role_id, is_active) VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'admin@sakura-dcim.local', '$2a$10$kgB.WoZJIqmBL1lm40rSCetQIaSFFmycu0hXC8nVSXK9CCHV4ytOO', 'System Admin', '00000000-0000-0000-0000-000000000001', true);
