# Sakura DCIM — Infrastructure Operating System

A production-grade **Infrastructure Operating System** for bare-metal data centers. Manages the full lifecycle of physical assets — from rack-mount discovery to automated OS provisioning — across geographically distributed sites.

**This is NOT a billing panel.** Sakura DCIM handles physical infrastructure only: servers, switches, IPAM, VLANs, and provisioning automation. Financial logic, customer portals, and invoicing are explicitly out of scope.

## Design Principles

| Principle | Implementation |
|-----------|---------------|
| **Idempotent provisioning** | Every provisioning step checks current state before acting; safe to retry |
| **Transactional safety** | Multi-step flows use status machines with rollback on failure |
| **Auditable** | Every state-changing operation logged with actor, timestamp, and diff |
| **Reversible** | IP unassign reverts switch port; PXE cleanup on completion/failure |
| **Driver abstraction** | Switch vendors behind a driver interface (Arista EOS, Cisco NX-OS/IOS, Juniper, etc.) |
| **Async-first** | NATS for task queuing; WebSocket for agent real-time communication |
| **Fail-open** | Optional subsystems (InfluxDB, Redis) degrade gracefully |

## Tech Stack

| Layer | Technology | Role |
|-------|-----------|------|
| Backend | Go 1.25 + Gin | API server, orchestration engine |
| Frontend | React 18 + TypeScript + Ant Design 5 + Vite | Operations console |
| Database | PostgreSQL 16 (pgx) | Source of truth — assets, IPAM, config |
| Cache / Lock | Redis 7 | Rate limiting, distributed locks, idempotency keys |
| Async Queue | NATS | Task queuing, provisioning pipeline (planned) |
| Time-Series | InfluxDB 2.x | Bandwidth counters, sensor history (optional) |
| Agent Comm | WebSocket + JWT | Real-time bidirectional agent control |
| Deployment | Docker Compose | Single-command dev/prod environment |
| State Mgmt | Zustand | Frontend reactive state |

## Architecture

```
                         ┌──────────────────────────┐
                         │     Operations Console    │
                         │   (React + Ant Design)    │
                         └───────────┬──────────────┘
                                     │ HTTPS
                         ┌───────────▼──────────────┐
                         │   Nginx (Reverse Proxy)   │
                         └──┬────────────────────┬──┘
                            │ /api               │ static
                 ┌──────────▼──────────────┐     │
                 │   Go Backend (Gin)      │     │
                 │                         │     │
                 │  ┌───────────────────┐  │     │
                 │  │ Provisioning      │  │     │
                 │  │ Engine            │  │     │
                 │  │  ├─ IP Assign     │  │     │
                 │  │  ├─ VLAN Push     │  │     │
                 │  │  ├─ PXE Boot      │  │     │
                 │  │  └─ Completion    │  │     │
                 │  ├───────────────────┤  │     │
                 │  │ IPAM             │  │     │
                 │  │  ├─ Pool Hierarchy│  │     │
                 │  │  ├─ Address Alloc │  │     │
                 │  │  └─ VRF / VLAN   │  │     │
                 │  ├───────────────────┤  │     │
                 │  │ Switch Automation │  │     │
                 │  │  ├─ Driver Layer  │  │     │
                 │  │  ├─ Port Provision│  │     │
                 │  │  └─ SNMP Sync    │  │     │
                 │  ├───────────────────┤  │     │
                 │  │ Auth / RBAC      │  │     │
                 │  │ Audit Logger     │  │     │
                 │  │ IPMI / KVM       │  │     │
                 │  └───────────────────┘  │     │
                 │                         │     │
                 │  ┌───────────────────┐  │     │
                 │  │ WebSocket Hub     │──┼─────┤
                 │  └──┬────┬────┬─────┘  │     │
                 └─────┼────┼────┼────────┘     │
                       │    │    │               │
            ┌──────────┘    │    └──────────┐    │
            ▼ WS            ▼ WS           ▼ WS │
   ┌────────────────┐ ┌────────────────┐ ┌────────────────┐
   │  Agent (DC-1)  │ │  Agent (DC-2)  │ │  Agent (DC-N)  │
   │                │ │                │ │                │
   │ ┌────────────┐ │ │                │ │                │
   │ │ Executors  │ │ │                │ │                │
   │ │ ├ IPMI     │ │ │  (same stack)  │ │  (same stack)  │
   │ │ ├ PXE/TFTP │ │ │                │ │                │
   │ │ ├ RAID     │ │ │                │ │                │
   │ │ ├ Switch   │ │ │                │ │                │
   │ │ ├ SNMP     │ │ │                │ │                │
   │ │ ├ KVM      │ │ │                │ │                │
   │ │ └ Inventory│ │ │                │ │                │
   │ └────────────┘ │ │                │ │                │
   └────────────────┘ └────────────────┘ └────────────────┘

   Data Stores:
   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
   │PostgreSQL│  │  Redis   │  │ InfluxDB │  │   NATS   │
   │ (state)  │  │ (lock)   │  │ (metrics)│  │ (queue)  │
   └──────────┘  └──────────┘  └──────────┘  └──────────┘
```

## Provisioning Engine — Core Flow

The provisioning engine is the heart of Sakura DCIM. It orchestrates the full server lifecycle from bare-metal to production-ready:

```
POST /servers/:id/provision
  │
  ├─ 1. Preflight Validation
  │     ├─ Agent online?
  │     ├─ MAC address present?
  │     ├─ Switch port linked?
  │     └─ IP already assigned?
  │
  ├─ 2. Status Transition → "provisioning"
  │
  ├─ 3. IP Assignment (idempotent)
  │     ├─ Skip if server already has IP
  │     ├─ Auto-select pool (by VRF or explicit pool_id)
  │     └─ Assign next available address
  │
  ├─ 4. Switch VLAN Push (automatic)
  │     ├─ Resolve pool → VLAN config (access/trunk/trunk_native)
  │     ├─ Find server's linked switch port
  │     └─ Agent SSH → configure port (vendor-specific driver)
  │
  ├─ 5. Network Config Resolution
  │     └─ Gateway, netmask, nameservers from IP pool
  │
  ├─ 6. PXE OS Install
  │     ├─ Render OS template (kickstart/preseed/autoinstall/cloud-init)
  │     │   └─ Inject: IP, gateway, netmask, DNS, MAC, hostname, SSH keys
  │     ├─ Agent: dnsmasq DHCP reservation (MAC → IP + DHCP options 1/3/6)
  │     ├─ Agent: TFTP kernel + initrd
  │     ├─ IPMI: set PXE boot device + reboot
  │     └─ Status → "reinstalling"
  │
  ├─ 7. Progress Tracking
  │     └─ Agent sends pxe.status events → real-time task updates
  │
  └─ 8. Completion
        ├─ PXE cleanup (remove dnsmasq/TFTP config)
        ├─ Post-install scripts execution
        └─ Status → "active" (or "error" on failure)
```

**Idempotency guarantees:**
- IP assignment checks existing allocations before allocating
- Install task guard prevents concurrent reinstalls on the same server
- Server status machine prevents re-entry (`provisioning` blocks new provision requests)
- PXE cleanup is fire-and-forget (safe to call multiple times)

## Switch Driver Abstraction

Switch automation uses a **driver pattern** to support multiple vendors:

| Vendor | Protocol | Status |
|--------|----------|--------|
| Cisco IOS/IOS-XE | SSH CLI | Implemented |
| Cisco NX-OS | SSH CLI | Implemented |
| Arista EOS | SSH CLI | Implemented |
| Juniper Junos | SSH CLI | Implemented |
| Dell Force10 | SSH CLI | Implemented |
| Huawei VRP | SSH CLI | Implemented |
| SONiC | SSH CLI | Implemented |
| Cumulus Linux | SSH CLI | Implemented |

**Driver capabilities:**
- Port mode configuration (access / trunk / trunk+native)
- VLAN assignment (fixed or auto-range allocation)
- Port admin toggle (up/down)
- DHCP relay configuration
- SNMP port discovery and sync
- Live port status query

**VLAN allocation strategies per IP pool:**
- `fixed` — Static VLAN ID from pool config
- `auto_range` — Next unused VLAN from pool's range (per-switch deduplication)

## IPAM (IP Address Management)

Hierarchical IP pool management with automatic switch integration:

```
Supernet /16 (type: subnet)
  ├─ Pool A /24 (type: ip_pool, VRF: production, VLAN: 100)
  │   ├─ 10.0.1.1  → assigned to server-01 (switch port auto-configured)
  │   ├─ 10.0.1.2  → assigned to server-02
  │   └─ 10.0.1.3  → available
  ├─ Pool B /24 (type: ip_pool, VRF: management, VLAN: 200)
  │   └─ ...
  └─ Subnet C /20 (type: subnet)
      └─ Pool D /28 (type: ip_pool)
```

**Features:**
- CIDR validation (child must be within parent range)
- Auto-generate host IPs from CIDR (skip network + broadcast)
- Gateway reservation
- VRF tagging
- Switch automation on IP assign/unassign (VLAN push/revert)
- Per-pool VLAN mode: access, trunk_native, trunk
- Per-pool nameservers (injected into PXE templates)

## Agent ↔ Backend Protocol

WebSocket + JSON with correlation IDs for request/response:

```json
{
  "id": "uuid-correlation",
  "type": "request | response | event",
  "action": "ipmi.power.status",
  "payload": { ... },
  "error": null
}
```

| Action | Direction | Description |
|--------|-----------|-------------|
| `agent.heartbeat` | Agent → Backend | Health + version |
| `ipmi.power.*` | Backend → Agent | Power on/off/reset/cycle/status |
| `ipmi.sensors` | Backend → Agent | Temperature, fan, voltage readings |
| `ipmi.kvm.start/stop` | Backend → Agent | KVM Docker container lifecycle |
| `pxe.prepare` | Backend → Agent | Configure dnsmasq + TFTP for PXE boot |
| `pxe.cleanup` | Backend → Agent | Remove PXE config after install |
| `pxe.status` | Agent → Backend | Install progress updates |
| `raid.configure` | Backend → Agent | RAID setup before OS install |
| `switch.provision` | Backend → Agent | Configure switch port (vendor-specific) |
| `switch.port_admin` | Backend → Agent | Toggle port up/down |
| `switch.status` | Backend → Agent | Query port state |
| `switch.dhcp_relay` | Backend → Agent | Configure DHCP relay on interface |
| `inventory.scan` | Backend → Agent | Hardware detection (CPU, RAM, disk, NIC) |
| `inventory.result` | Agent → Backend | Hardware scan results |
| `snmp.poll` | Backend → Agent | Bandwidth counter collection |
| `discovery.start/stop` | Backend → Agent | PXE discovery session |
| `discovery.result` | Agent → Backend | Discovered server report |

## KVM Architecture (Docker Browser Isolation)

Universal BMC console access via Docker-isolated Chromium:

```
Browser (noVNC) ←─WS─→ Backend (relay) ←─WS─→ Agent ←─TCP─→ Docker (Chromium+VNC) → BMC Web UI
```

- **Universal**: Works with iDRAC, iLO, Supermicro, ASRock, Lenovo XCC, Huawei iBMC
- **Secure**: Users never access BMC directly; credentials injected into isolated container
- **Per-session**: ~200-400MB RAM, auto-destroyed on disconnect or 30-min timeout

## Project Structure

```
sakura-dcim/
├── backend/                          # Go API + orchestration engine
│   ├── cmd/server/main.go            # Entry point, service wiring
│   ├── internal/
│   │   ├── config/                   # Viper configuration
│   │   ├── domain/                   # Entity models, enums, value objects
│   │   │   ├── server.go             # Server, ServerStatus, BMCType
│   │   │   ├── network.go            # Switch, SwitchPort, IPPool, IPAddress
│   │   │   ├── os_profile.go         # OSProfile, DiskLayout, InstallTask, ProvisionRequest
│   │   │   ├── switch_templates.go   # Vendor CLI command templates
│   │   │   ├── tenant.go             # Tenant hierarchy
│   │   │   ├── user.go / role.go     # RBAC
│   │   │   ├── audit_log.go          # Audit trail
│   │   │   ├── discovery.go          # PXE discovery
│   │   │   └── common.go             # APIResponse, PaginatedResult
│   │   ├── handler/                  # HTTP handlers (thin, delegate to services)
│   │   ├── middleware/               # Auth, RBAC, Audit, RateLimit, Prometheus, Security
│   │   ├── repository/              # Interface definitions + PostgreSQL/InfluxDB impls
│   │   │   ├── repository.go         # All repository interfaces
│   │   │   ├── postgres/             # 17 PostgreSQL implementations
│   │   │   └── influxdb/             # Bandwidth time-series
│   │   ├── service/                  # Business logic (20 services)
│   │   │   ├── provision_service.go  # ★ Core: orchestrates full provisioning flow
│   │   │   ├── reinstall_service.go  # PXE template rendering + install tracking
│   │   │   ├── ip_service.go         # IPAM: pools, addresses, auto-assign, switch automation
│   │   │   ├── switch_service.go     # Port provisioning, SNMP sync, driver dispatch
│   │   │   ├── ipmi_service.go       # Power control, sensors
│   │   │   ├── kvm_service.go        # KVM session management
│   │   │   ├── bandwidth_service.go  # SNMP → InfluxDB, 95th percentile
│   │   │   └── ...                   # Auth, agent, inventory, discovery, etc.
│   │   ├── websocket/               # Agent WS hub + protocol
│   │   └── pkg/crypto/              # AES-256-GCM, JWT, bcrypt
│   └── migrations/                   # SQL schema (13 versioned migrations)
│
├── agent/                            # Lightweight Go agent (per datacenter)
│   ├── cmd/agent/main.go
│   └── internal/
│       ├── client/                   # WebSocket client + auto-reconnect
│       ├── config/                   # YAML config + hot-reload
│       └── executor/                 # 11 executors
│           ├── pxe.go                # dnsmasq + TFTP + DHCP options
│           ├── raid.go               # storcli + mdadm
│           ├── switch.go             # SSH + vendor CLI templates
│           ├── ipmi.go               # Power + sensors (vendor-aware cipher suites)
│           ├── kvm.go                # Docker Chromium + VNC relay
│           ├── snmp.go               # Interface counter polling
│           ├── inventory.go          # lscpu, dmidecode, lsblk, ip addr
│           ├── discovery.go          # PXE boot unknown servers
│           └── ...                   # SOL, PXE inventory, IPMI user mgmt
│
├── web/                              # React operations console
│   └── src/
│       ├── api/                      # Typed API client (Axios + interceptors)
│       ├── components/Layout/        # App shell, sidebar, branding
│       ├── pages/                    # ~15 pages
│       │   ├── Servers/              # Server list + detail (10 tabs)
│       │   │   └── tabs/             # Overview, Power, Sensors, KVM, Provision,
│       │   │                         # Reinstall, Bandwidth, Network, Inventory, IP
│       │   ├── Switches/             # Switch + port management
│       │   ├── IPPools/              # Hierarchical IPAM
│       │   └── ...                   # Agents, Users, Roles, Tenants, Audit, etc.
│       ├── store/                    # Zustand (auth, branding)
│       └── types/                    # Full TypeScript interfaces
│
├── docker/
│   ├── kvm-browser/                  # KVM Docker image (Chromium + VNC)
│   └── backup/                       # PostgreSQL pg_dump cron
├── docker-compose.yml
└── Makefile
```

## Database Schema

17 tables across 13 versioned migrations:

```
tenants (hierarchical: admin → reseller → customer)
  └── users ──► roles (25+ permissions, RBAC)

agents (per-datacenter, WebSocket-connected)
  └── servers
        ├── server_inventory (CPU, RAM, disk, NIC components)
        ├── server_disks (for RAID configuration)
        └── install_tasks ──► os_profiles
                              ├── disk_layouts
                              └── scripts (post-install, ordered)

switches (SSH + SNMP credentials, vendor type)
  └── switch_ports ──► servers
        ├── port_mode: access / trunk / trunk_native
        ├── vlan_id, native_vlan_id, trunk_vlans
        └── auto-provisioned on IP assignment

ip_pools (hierarchical: parent_id self-ref)
  ├── pool_type: ip_pool (leaf) / subnet (container)
  ├── vlan_mode, vlan_allocation (fixed / auto_range)
  ├── vrf, gateway, netmask, nameservers
  ├── ip_pools (children)
  └── ip_addresses ──► servers
        └── status: available / assigned / reserved

audit_logs (append-only, all mutations)

discovery_sessions ──► discovered_servers
  └── approve → creates server with pre-filled hardware specs
```

## Server Status Machine

```
           ┌─────────────────────────────────────┐
           │                                     │
           ▼                                     │
  ┌─────────────┐   provision   ┌──────────────┐ │
  │   active    │──────────────►│ provisioning │ │
  └─────────────┘               └──────┬───────┘ │
         ▲                             │         │
         │                    PXE boot │         │
         │                             ▼         │
         │                    ┌──────────────┐   │
         │    completed       │ reinstalling │   │
         │◄───────────────────┤              │   │
         │                    └──────┬───────┘   │
         │                           │           │
         │                   failure │           │
         │                           ▼           │
         │                    ┌──────────────┐   │
         │                    │    error     │───┘
         │                    └──────────────┘
         │                           │ retry
         │                           │
         └───────────────────────────┘

  offline ◄──── agent heartbeat timeout
```

## Security

| Layer | Mechanism |
|-------|-----------|
| **Authentication** | JWT access (15min) + refresh (7d, HttpOnly cookie) |
| **Authorization** | 25+ permissions, middleware-enforced, tenant-scoped |
| **Tenant Isolation** | Every query filtered by tenant_id |
| **Credential Storage** | AES-256-GCM encryption at rest (IPMI, switch SSH/SNMP) |
| **Agent Auth** | Per-agent tokens, bcrypt hashed |
| **KVM Isolation** | Docker browser container, user never touches BMC directly |
| **Rate Limiting** | Redis sliding window (auth endpoints) |
| **Security Headers** | CSP, X-Frame-Options, HSTS, X-Content-Type-Options |
| **Audit Trail** | Every mutation logged (actor, IP, user-agent, sanitized body) |

## API Endpoints

```
Auth:
  POST   /api/v1/auth/login
  POST   /api/v1/auth/refresh
  GET    /api/v1/auth/me
  POST   /api/v1/auth/logout
  GET    /api/v1/auth/branding          # Public, tenant branding by domain/slug

Servers:
  CRUD   /api/v1/servers
  POST   /api/v1/servers/:id/power      # IPMI power control
  GET    /api/v1/servers/:id/power      # Power status
  GET    /api/v1/servers/:id/sensors    # IPMI sensors
  POST   /api/v1/servers/:id/kvm       # Start KVM session
  DELETE /api/v1/servers/:id/kvm       # Stop KVM session
  GET    /api/v1/servers/:id/provision/preflight   # Pre-flight check
  POST   /api/v1/servers/:id/provision             # Full provisioning flow
  POST   /api/v1/servers/:id/reinstall             # OS reinstall only
  GET    /api/v1/servers/:id/reinstall/status      # Install progress
  GET    /api/v1/servers/:id/bandwidth             # Bandwidth stats
  GET    /api/v1/servers/:id/inventory             # Hardware inventory
  POST   /api/v1/servers/:id/inventory/scan        # Trigger scan

KVM WebSocket:
  WS     /api/v1/kvm/ws                # Browser noVNC
  WS     /api/v1/kvm/relay             # Agent VNC relay

Agents:
  CRUD   /api/v1/agents
  WS     /api/v1/agents/ws             # Agent WebSocket

Discovery:
  POST   /api/v1/agents/:id/discovery/start
  POST   /api/v1/agents/:id/discovery/stop
  GET    /api/v1/agents/:id/discovery/status
  GET    /api/v1/discovery/servers
  POST   /api/v1/discovery/servers/:id/approve
  POST   /api/v1/discovery/servers/:id/reject

Switches:
  CRUD   /api/v1/switches
  CRUD   /api/v1/switches/:id/ports
  POST   /api/v1/switches/:id/ports/:portId/provision
  GET    /api/v1/switches/:id/ports/:portId/status
  POST   /api/v1/switches/:id/ports/:portId/admin
  POST   /api/v1/switches/:id/sync-ports
  POST   /api/v1/switches/:id/test
  POST   /api/v1/switches/:id/snmp-poll
  GET    /api/v1/switches/:id/vlans
  GET    /api/v1/switches/templates
  GET    /api/v1/switches/server/:serverId/ports
  PUT    /api/v1/switches/ports/:portId/link
  PUT    /api/v1/switches/ports/:portId/unlink

IPAM:
  CRUD   /api/v1/ip-pools
  CRUD   /api/v1/ip-pools/:id/addresses
  POST   /api/v1/ip-pools/:id/assign
  GET    /api/v1/ip-pools/:id/children
  POST   /api/v1/ip-pools/:id/generate
  GET    /api/v1/ip-pools/assignable
  GET    /api/v1/ip-pools/by-server/:serverId
  POST   /api/v1/ip-pools/auto-assign

Resources:
  CRUD   /api/v1/os-profiles
  CRUD   /api/v1/disk-layouts
  CRUD   /api/v1/scripts
  CRUD   /api/v1/users
  CRUD   /api/v1/roles
  CRUD   /api/v1/tenants
  GET    /api/v1/tenants/:id/children
  GET    /api/v1/tenants/:id/tree
  GET    /api/v1/settings
  PUT    /api/v1/settings

Audit:
  GET    /api/v1/audit-logs

Monitoring:
  GET    /metrics                        # Prometheus
```

## Implementation Status

### Completed

| Phase | Scope |
|-------|-------|
| 1 | Foundation — Auth, RBAC, multi-tenant, Docker |
| 2 | Server CRUD, agent WebSocket, management pages |
| 3 | IPMI power control + sensor monitoring |
| 4 | NoVNC KVM console (Docker browser isolation) |
| 5 | PXE OS reinstall + auto RAID + post-install scripts |
| 6 | Switch automation + bandwidth monitoring (8 vendors) |
| 7 | Hardware inventory + hierarchical IPAM |
| 8 | White-label + multi-tenant branding |
| 9 | Audit hardening + security headers + Prometheus |
| 10 | Provisioning engine (IP → VLAN → PXE orchestration) |

### Architecture Evolution Roadmap

The following items evolve Sakura DCIM from a management panel into a true **Infrastructure Operating System**:

| Priority | Item | Description |
|----------|------|-------------|
| P0 | **NATS task queue** | Replace goroutine-based async dispatch with NATS JetStream for durable, retryable provisioning tasks |
| P0 | **Idempotency keys** | Redis-backed idempotency keys for all provisioning API calls (prevent duplicate execution) |
| P0 | **Distributed locks** | Redis SETNX locks for per-server provisioning (prevent concurrent operations on same asset) |
| P1 | **Switch driver interface** | Extract vendor CLI generation from agent templates into a proper `SwitchDriver` interface with per-vendor implementations |
| P1 | **Provisioning state machine** | Formal state machine (saga pattern) with compensating transactions for each provisioning step |
| P1 | **Transaction wrapping** | PostgreSQL transactions around multi-step operations (IP assign + server update + audit log) |
| P2 | **NATS event bus** | Publish domain events (server.provisioned, ip.assigned, port.configured) for decoupled consumers |
| P2 | **Retry policies** | Configurable retry with exponential backoff for agent commands (switch SSH, IPMI, PXE) |
| P2 | **Rollback engine** | Automatic compensation on provisioning failure (release IP, revert switch port, cleanup PXE) |
| P3 | **IPv6 IPAM** | Dual-stack support in pool management, address generation, and PXE templates |
| P3 | **Prefix delegation** | Automated /48 → /64 prefix allocation for IPv6 pools |

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 20+
- Docker & Docker Compose

### Development

```bash
# 1. Start infrastructure (PostgreSQL, Redis, InfluxDB)
make docker-infra

# 2. Run database migrations
make migrate

# 3. Build KVM browser image
docker build -t sakura-dcim/kvm-browser:latest docker/kvm-browser/

# 4. Start backend (terminal 1)
make dev-backend
# → API at http://localhost:8080

# 5. Start frontend (terminal 2)
make dev-web
# → Console at http://localhost:5173

# 6. Default credentials
#    Email:    admin@sakura-dcim.local
#    Password: admin123
```

### Production (Docker Compose)

```bash
docker compose up -d
# Console at http://localhost:3000
# API at http://localhost:3000/api/v1
```

### Deploy an Agent

```bash
# 1. In the console: Agents → Add Agent → copy token
# 2. On the datacenter server:
cp agent/config.yaml.example agent/config.yaml
# Edit with panel URL, agent ID, and token

# 3. Run
cd agent && go run ./cmd/agent
```

## Make Commands

```bash
make dev-backend     # Run Go backend (dev mode)
make dev-web         # Run React frontend (dev mode)
make build-backend   # Build backend binary
make build-agent     # Build agent binary
make build-web       # Build frontend
make migrate         # Run database migrations
make migrate-down    # Rollback last migration
make docker-infra    # Start PG + Redis + InfluxDB
make docker-up       # Start all services
make docker-down     # Stop all services
make test-backend    # Run Go tests
make lint-backend    # Run Go linter
make build-kvm       # Build KVM Docker image
make clean           # Remove build artifacts
```

## License

Private — All rights reserved.

## AI Rules

### Must Follow
1. **Read before edit**: Always Read a file before modifying it
2. **Verify compilation** after every change
3. **No breaking changes**: All existing imports must continue to work
4. **Single module iteration**: Complete one file before moving to the next
5. **JSON snake_case**: Backend API responses use snake_case field names
6. **Components < 150 lines**: Extract sub-components when exceeded
7. **Preserve comments**: Keep all existing comments when moving code
8. **No reasoning output**: Generate internally, output only final code
9. **Understand context**: Read full project if no memory exists
10. **Read this file** at every new session start
11. **Auto-commit**: After confirming a feature works, auto-generate commit message and commit
12. **Versioned migrations**: All database changes via numbered migration files

### Prohibited
- No hardcoded colors/spacing — use CSS variables or Ant Design 5 tokens
- No business logic in handlers — delegate to service layer
- No direct repository calls — always go through service layer
- No duplicate components — check existing before creating
- No over-engineering — no abstractions for hypothetical future needs
- No unsolicited docstrings, comments, or type annotations
- No speculation — no repeated analysis loops
- No explanations — output final code only
- No billing logic — no customers, no invoicing, no financial operations
