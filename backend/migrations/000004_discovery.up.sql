-- Discovery sessions (one active per agent at a time)
CREATE TABLE discovery_sessions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id       UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    status         VARCHAR(20) NOT NULL DEFAULT 'active',
    callback_token VARCHAR(64) NOT NULL,
    dhcp_range     VARCHAR(100) NOT NULL DEFAULT '',
    started_by     UUID REFERENCES users(id),
    started_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at     TIMESTAMPTZ
);
CREATE INDEX idx_discovery_sessions_agent ON discovery_sessions(agent_id);

-- Discovered servers
CREATE TABLE discovered_servers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES discovery_sessions(id) ON DELETE CASCADE,
    agent_id        UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    mac_address     VARCHAR(20) NOT NULL,
    ip_address      VARCHAR(45),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    system_vendor   VARCHAR(255) NOT NULL DEFAULT '',
    system_product  VARCHAR(255) NOT NULL DEFAULT '',
    system_serial   VARCHAR(255) NOT NULL DEFAULT '',
    cpu_model       VARCHAR(255) NOT NULL DEFAULT '',
    cpu_cores       INT NOT NULL DEFAULT 0,
    cpu_sockets     INT NOT NULL DEFAULT 0,
    ram_mb          BIGINT NOT NULL DEFAULT 0,
    disk_count      INT NOT NULL DEFAULT 0,
    disk_total_gb   BIGINT NOT NULL DEFAULT 0,
    nic_count       INT NOT NULL DEFAULT 0,
    raw_inventory   JSONB NOT NULL DEFAULT '{}',
    bmc_ip          VARCHAR(45) NOT NULL DEFAULT '',
    approved_by     UUID REFERENCES users(id),
    server_id       UUID REFERENCES servers(id),
    discovered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, mac_address)
);
CREATE INDEX idx_discovered_servers_agent ON discovered_servers(agent_id);
CREATE INDEX idx_discovered_servers_status ON discovered_servers(status);
