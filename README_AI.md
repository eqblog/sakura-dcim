# Sakura DCIM

A production-grade Data Center Infrastructure Management platform. Manage all your dedicated servers across multiple datacenters from a single, clean web interface.

Inspired by [Tenantos](https://tenantos.com/) and [EasyDCIM](https://www.easydcim.com/).

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25 + Gin |
| Frontend | React 18 + TypeScript + Ant Design 5 + Vite |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Time-Series | InfluxDB 2.x |
| Agent Comm | WebSocket + JWT |
| Deployment | Docker Compose |

## Architecture

```
                        ┌─────────────────────────┐
                        │      Web Browser         │
                        │  (React + Ant Design)    │
                        └────────────┬────────────┘
                                     │ HTTPS
                        ┌────────────▼────────────┐
                        │     Nginx (Reverse Proxy)│
                        └──┬──────────────────┬───┘
                           │ /api             │ static
                ┌──────────▼──────────┐       │
                │   Go Backend (Gin)  │       │
                │                     │       │
                │  ┌───────────────┐  │       │
                │  │ Auth/RBAC     │  │       │
                │  │ Server Mgmt   │  │       │
                │  │ IPMI Service  │  │       │
                │  │ PXE Service   │  │       │
                │  │ KVM Proxy     │  │       │
                │  │ Bandwidth     │  │       │
                │  │ Audit Logger  │  │       │
                │  └───────────────┘  │       │
                │                     │       │
                │  ┌───────────────┐  │       │
                │  │ WebSocket Hub │  │       │
                │  └──┬───┬───┬───┘  │       │
                └─────┼───┼───┼──────┘       │
                      │   │   │              │
            ┌─────────┘   │   └─────────┐    │
            ▼ WS          ▼ WS          ▼ WS │
   ┌────────────┐  ┌────────────┐  ┌────────────┐
   │ Agent DC-1 │  │ Agent DC-2 │  │ Agent DC-N │
   │            │  │            │  │            │
   │ • ipmitool │  │ • ipmitool │  │ • ipmitool │
   │ • PXE/TFTP │  │ • PXE/TFTP │  │ • PXE/TFTP │
   │ • KVM proxy│  │ • KVM proxy│  │ • KVM proxy│
   │ • SNMP poll│  │ • SNMP poll│  │ • SNMP poll│
   │ • Inventory│  │ • Inventory│  │ • Inventory│
   └────────────┘  └────────────┘  └────────────┘

   Data Stores:
   ┌──────────┐  ┌──────────┐  ┌──────────┐
   │PostgreSQL│  │  Redis   │  │ InfluxDB │
   │ (CRUD)   │  │ (Cache)  │  │ (Metrics)│
   └──────────┘  └──────────┘  └──────────┘
```

## Features

### Implemented (Phase 1 — Foundation)

- **JWT Authentication** — Access + Refresh token flow with auto-refresh
- **RBAC** — 25+ granular permissions, 3 built-in roles (Super Admin / Admin / Customer)
- **Multi-Tenant** — Admin → Reseller → Customer hierarchy with resource isolation
- **Server CRUD** — Create, list, search, filter, paginate servers
- **Agent WebSocket Hub** — Connection management, heartbeat, request/response protocol
- **Agent Binary** — IPMI executor, inventory scanner, auto-reconnect
- **Audit Logging** — All state-changing API calls logged with IP, user-agent, details
- **React Dashboard** — Server stats, agent status, recent servers
- **Full API Client** — Axios with interceptors, token refresh, TypeScript types
- **Docker Compose** — One-command dev environment

### Planned

| Feature | Description |
|---------|-------------|
| **IPMI Power Control** | On / Off / Reset / Cycle via agent-proxied ipmitool |
| ~~**NoVNC KVM Console**~~ | ✅ Implemented — Docker browser isolation (Chromium kiosk → BMC web UI), proxied via noVNC |
| **PXE OS Reinstall** | One-click unattended OS reinstallation via PXE boot |
| **Auto RAID** | Automatic RAID 1/5/10 based on disk count, or customer-selected |
| **Disk Layouts** | Custom partition table templates per server tag or OS profile |
| **Post-Install Scripts** | Shell scripts executed after OS installation |
| **SNMP Bandwidth** | Switch port monitoring, traffic stats, 95th percentile billing |
| **IPMI Sensor Graphs** | Temperature, fan speed, voltage charts via InfluxDB |
| **Hardware Inventory** | Auto-detect CPU, RAM, disks, NICs via PXE boot or shell command |
| **IP Pool Management** | CIDR-based IP pools with assignment tracking |
| **White-Label** | Custom domain, logo, colors, favicon per tenant |
| **Reseller System** | Unlimited nesting, sub-resellers, per-reseller branding |

## Project Structure

```
sakura-dcim/
├── backend/                    # Go API server
│   ├── cmd/server/main.go      # Entry point
│   ├── internal/
│   │   ├── config/             # Viper configuration
│   │   ├── domain/             # Entity models (8 files)
│   │   ├── handler/            # Gin HTTP handlers
│   │   ├── middleware/         # Auth, RBAC, Audit, Rate-limit
│   │   ├── repository/        # PostgreSQL implementations
│   │   ├── service/           # Business logic
│   │   ├── websocket/         # Agent WS hub + protocol
│   │   └── pkg/crypto/        # AES-256-GCM, JWT, bcrypt
│   └── migrations/            # SQL schema (17 tables)
│
├── agent/                      # Lightweight Go agent (per datacenter)
│   ├── cmd/agent/main.go       # Entry point
│   └── internal/
│       ├── client/             # WebSocket client + reconnect
│       ├── config/             # YAML config
│       └── executor/           # IPMI, Inventory executors
│
├── web/                        # React frontend
│   └── src/
│       ├── api/                # Axios client + all API endpoints
│       ├── components/Layout/  # App shell with sidebar
│       ├── pages/              # 12 pages (Dashboard, Servers, Agents, ...)
│       ├── store/              # Zustand auth store
│       └── types/              # Full TypeScript interfaces
│
├── docker/                     # Dockerfiles + nginx config
├── docker-compose.yml          # Full dev environment
└── Makefile                    # Dev, build, migrate, test commands
```

## Database Schema

17 tables organized around the core entities:

```
tenants (multi-tenant hierarchy)
  └── users ──► roles (RBAC, 25+ permissions)

agents (remote datacenter agents)
  └── servers
        ├── server_inventory (hardware details)
        ├── server_disks (for RAID operations)
        └── install_tasks ──► os_profiles
                              ├── disk_layouts
                              └── scripts (post-install)

switches (SNMP)
  └── switch_ports ──► servers

ip_pools
  └── ip_addresses ──► servers

audit_logs (all mutations logged)
```

## Agent ↔ Panel Protocol

WebSocket + JSON, request/response with correlation IDs:

```json
{
  "id": "uuid",
  "type": "request | response | event",
  "action": "ipmi.power.status",
  "payload": { "ipmi_ip": "10.0.0.1", "ipmi_user": "ADMIN", "ipmi_pass": "..." },
  "error": null
}
```

| Action | Direction | Description |
|--------|-----------|-------------|
| `agent.heartbeat` | Agent → Panel | Periodic health + version info |
| `ipmi.power.*` | Panel → Agent | Power on/off/reset/cycle/status |
| `ipmi.sensors` | Panel → Agent | Read temperature, fan, voltage |
| `ipmi.kvm.start` | Panel → Agent | Start KVM proxy session |
| `pxe.prepare` | Panel → Agent | Configure DHCP/TFTP for reinstall |
| `pxe.status` | Agent → Panel | Installation progress updates |
| `raid.configure` | Panel → Agent | Set up RAID before OS install |
| `inventory.scan` | Panel → Agent | Trigger hardware detection |
| `snmp.poll` | Panel → Agent | Poll switch bandwidth counters |

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 20+
- Docker & Docker Compose

### Development

```bash
# 1. Start infrastructure (PostgreSQL, Redis, InfluxDB)
make docker-infra

# 2. Run database migrations
make migrate

# 3. Start backend (terminal 1)
make dev-backend
# → API server at http://localhost:8080

# 4. Start frontend (terminal 2)
make dev-web
# → Web UI at http://localhost:5173

# 5. Login
#    Email:    admin@sakura-dcim.local
#    Password: admin123
```

### Production (Docker Compose)

```bash
# Build and start all services
docker compose up -d

# Web UI at http://localhost:3000
# API at http://localhost:3000/api/v1
```

### Deploy an Agent

```bash
# 1. In the web panel, go to Agents → Add Agent
# 2. Copy the generated token
# 3. On the datacenter server:

cp agent/config.yaml.example agent/config.yaml
# Edit config.yaml with your panel URL, agent ID, and token

# 4. Run the agent
cd agent && go run ./cmd/agent
# Or build and deploy as a systemd service
```

## API Endpoints

```
Auth:
  POST   /api/v1/auth/login          # Login
  POST   /api/v1/auth/refresh        # Refresh token
  GET    /api/v1/auth/me             # Current user
  POST   /api/v1/auth/logout         # Logout

Servers:
  GET    /api/v1/servers              # List (search, filter, paginate)
  POST   /api/v1/servers              # Create
  GET    /api/v1/servers/:id          # Detail
  PUT    /api/v1/servers/:id          # Update
  DELETE /api/v1/servers/:id          # Delete
  POST   /api/v1/servers/:id/power    # Power control
  GET    /api/v1/servers/:id/sensors  # IPMI sensors
  GET    /api/v1/servers/:id/kvm      # KVM console URL
  POST   /api/v1/servers/:id/reinstall # OS reinstall
  GET    /api/v1/servers/:id/bandwidth # Bandwidth stats

Agents:
  GET    /api/v1/agents               # List
  POST   /api/v1/agents               # Register
  WS     /api/v1/agents/ws            # Agent WebSocket

Resources:
  CRUD   /api/v1/os-profiles          # OS templates
  CRUD   /api/v1/disk-layouts         # Partition templates
  CRUD   /api/v1/scripts              # Post-install scripts
  CRUD   /api/v1/switches             # SNMP switches
  CRUD   /api/v1/ip-pools             # IP address pools
  CRUD   /api/v1/users                # User management
  CRUD   /api/v1/roles                # Role management
  CRUD   /api/v1/tenants              # Tenant management

Audit:
  GET    /api/v1/audit-logs           # Search & filter
```

## Implementation Roadmap

| Phase | Scope | Status |
|-------|-------|--------|
| 1 | Foundation — Auth, RBAC, DB, Layout, Docker | ✅ Done |
| 2 | Server CRUD + Agent WebSocket wiring | ✅ Done |
| 3 | IPMI Power Control + Sensor Monitoring | 🔲 Next |
| 4 | NoVNC KVM Console (Docker Browser Isolation) | ✅ Done |
| 5 | PXE OS Reinstall + Auto RAID + Scripts | 🔲 |
| 6 | SNMP Bandwidth Monitoring + Charts | 🔲 |
| 7 | Hardware Inventory + IP Management | 🔲 |
| 8 | White-Label + Multi-Tenant Polish | 🔲 |
| 9 | Audit Hardening + API Docs + Security | 🔲 |

## Detailed TODO

### Phase 2 — Server & Agent Management ✅
- [x] `backend` Agent CRUD handler + service (POST/GET/PUT/DELETE /agents)
- [x] `backend` Agent WebSocket heartbeat → update last_seen + status in DB
- [x] `backend` Agent event router (dispatch PXE status, SNMP data to services)
- [x] `backend` Tenant CRUD handler + service
- [x] `backend` User CRUD handler + service (with password hashing)
- [x] `backend` Role CRUD handler + service
- [x] `web` Server detail page with tabs (Overview / Power / KVM / Reinstall / Bandwidth / Inventory)
- [x] `web` Agent detail page with live status + capabilities
- [x] `web` User management page (list, create, edit, delete)
- [x] `web` Role management page with permission checkboxes
- [x] `web` Tenant management page
- [ ] `agent` Config hot-reload support (deferred)

### Phase 3 — IPMI & Power Management
- [ ] `backend` IPMI service: power control via agent WebSocket (decrypt creds → send to agent)
- [ ] `backend` IPMI handler: POST /servers/:id/power, GET /servers/:id/power
- [ ] `backend` IPMI sensor handler: GET /servers/:id/sensors
- [ ] `backend` Sensor data collector → write to InfluxDB on interval
- [ ] `backend` InfluxDB repository: write sensor readings, query time-series
- [ ] `web` Server Power tab: buttons (On/Off/Reset/Cycle) + live status badge
- [ ] `web` Server Sensors tab: real-time sensor table + temperature/fan/voltage charts
- [ ] `agent` IPMI executor: SOL (Serial Over LAN) support

### Phase 4 — NoVNC KVM Console (Docker Browser Isolation) ✅
- [x] `docker` KVM browser image: Alpine + Xvfb + x11vnc + Chromium kiosk mode
- [x] `agent` KVM executor: Docker container lifecycle + VNC TCP↔WebSocket relay
- [x] `backend` KVM service: session management, IPMI credential decryption, agent dispatch
- [x] `backend` KVM handler: POST /servers/:id/kvm + WebSocket relay (browser↔backend↔agent)
- [x] `backend` KVM WebSocket proxy: /kvm/ws (browser) + /kvm/relay (agent) dual-endpoint
- [x] `web` KVM Console page: noVNC integration, fullscreen toggle, auto-disconnect on close

### Phase 5 — PXE & OS Reinstallation
- [ ] `backend` OS Profile CRUD handler + service
- [ ] `backend` Disk Layout CRUD handler + service
- [ ] `backend` Script CRUD handler + service
- [ ] `backend` Install task workflow: create → pxe_booting → installing → post_scripts → completed/failed
- [ ] `backend` Template engine: render Kickstart/Preseed/Autoinstall/cloud-init with server variables
- [ ] `backend` Auto RAID logic: pick RAID level based on disk count (1 disk=none, 2=RAID1, 3+=RAID5, 4+=RAID10)
- [ ] `agent` PXE executor: manage dnsmasq DHCP/TFTP config per server MAC
- [ ] `agent` RAID executor: storcli/megacli (HW RAID) + mdadm (SW RAID)
- [ ] `agent` Post-install callback: report progress via WebSocket event
- [ ] `web` OS Profiles page: CRUD with template editor (CodeMirror/Monaco)
- [ ] `web` Disk Layouts page: visual partition editor
- [ ] `web` Scripts page: CRUD with shell editor
- [ ] `web` Reinstall Wizard: Select OS → RAID config → Disk layout → SSH keys → Review → Install
- [ ] `web` Install progress: real-time progress bar + log stream

### Phase 6 — Bandwidth Monitoring
- [ ] `backend` Switch CRUD handler + service
- [ ] `backend` Switch port mapping handler (assign port → server)
- [ ] `backend` Bandwidth service: query InfluxDB for traffic data
- [ ] `backend` 95th percentile calculation endpoint
- [ ] `agent` SNMP executor: periodic polling of ifInOctets/ifOutOctets via gosnmp
- [ ] `agent` SNMP data → send to panel via WebSocket event → panel writes to InfluxDB
- [ ] `web` Switches page: CRUD + port mapping table
- [ ] `web` Server Bandwidth tab: in/out traffic chart (hourly/daily/monthly), 95th line
- [ ] `web` Bandwidth overview page: top talkers, aggregate stats

### Phase 7 — Inventory & IP Management
- [ ] `backend` Inventory handler: GET /servers/:id/inventory, POST /servers/:id/inventory/scan
- [ ] `backend` Inventory service: store scan results, parse components
- [ ] `backend` IP Pool CRUD handler + service
- [ ] `backend` IP Address assignment/release logic (auto-assign from pool)
- [ ] `agent` Inventory scanner: lshw JSON → parse CPU/RAM/Disk/NIC/GPU
- [ ] `agent` PXE inventory mode: boot to mini-Linux, scan, report, reboot
- [ ] `web` Server Inventory tab: component tree (CPU, RAM DIMMs, Disks, NICs)
- [ ] `web` IP Pools page: pool CRUD + address table with status
- [ ] `web` IP assignment modal in server detail

### Phase 8 — White-Label & Multi-Tenant Polish
- [ ] `backend` Tenant settings API: logo, colors, favicon, custom domain
- [ ] `backend` Custom domain middleware: resolve domain → tenant
- [ ] `backend` Reseller hierarchy: nested tenant tree, cascading permissions
- [ ] `backend` Tenant-scoped statistics API
- [ ] `web` Dynamic theme: load tenant branding on login (CSS variables)
- [ ] `web` Tenant settings page: upload logo, pick colors, set domain
- [ ] `web` Reseller dashboard: manage sub-tenants, assign servers
- [ ] `web` Hide branding completely (no "Sakura DCIM" references in white-label mode)

### Phase 9 — Audit, Logging & Hardening
- [ ] `backend` Audit middleware: capture request body (sanitize passwords)
- [ ] `backend` System event log: agent connect/disconnect, install failures, IPMI errors
- [ ] `backend` Rate limiter: per-user on auth + power endpoints (go-redis sliding window)
- [ ] `backend` CSP/CORS/HSTS security headers
- [ ] `backend` Input validation: sanitize all user inputs
- [ ] `backend` Swagger/OpenAPI generation via swaggo
- [ ] `web` Audit log page: advanced filters (user, action, resource, date range, IP)
- [ ] `web` System event timeline
- [ ] `web` API documentation page (embed Swagger UI)
- [ ] `infra` Automated backups (PostgreSQL pg_dump cron)
- [ ] `infra` Health check endpoints for all services
- [ ] `infra` Prometheus metrics exporter

## Security

- **JWT** — Short-lived access tokens (15min) + refresh tokens (7d, HttpOnly cookie)
- **RBAC** — 25+ permissions, middleware-enforced, tenant-scoped
- **Tenant Isolation** — Every query filtered by tenant_id
- **IPMI Credentials** — AES-256-GCM encrypted at rest, decrypted only when sent to agent
- **Agent Auth** — Per-agent tokens, bcrypt hashed, validated on WebSocket connect
- **KVM Sessions** — Short-lived tokens (5min), agent proxies so IPMI stays on private network
- **Audit Trail** — Every mutation logged with user, IP, user-agent, request details

## Make Commands

```bash
make dev-backend     # Run Go backend in dev mode
make dev-web         # Run React frontend in dev mode
make build-backend   # Build Go backend binary
make build-agent     # Build Go agent binary
make build-web       # Build React frontend
make migrate         # Run database migrations
make migrate-down    # Rollback last migration
make docker-infra    # Start PG + Redis + InfluxDB
make docker-up       # Start all services
make docker-down     # Stop all services
make test-backend    # Run Go tests
make lint-backend    # Run Go linter
make clean           # Remove build artifacts
```

## License

Private — All rights reserved.

## AI 操作规则

### 必须遵守
1. **先读后改**: 修改文件前必须先 Read
2. **每步验证编译**
3. **不破坏兼容**: 所有现有 import 必须继续工作 (用 barrel re-export)
4. **单模块迭代**: 完成一个文件的重构再进行下一个
5. **JSON snake_case**: 后端 API 响应使用 snake_case 字段名
6. **组件 <150 行**: 前端提取的子组件目标行数
7. **保留注释**: 移动代码时保留所有现有注释
8. **生成前先内部推理，但不要输出推理过程**
9. **加强联系上下文** 必须要理解整个项目，如果没有记忆请读取整个项目并理解
10. **每次打开新的session请必须阅读本文档AI_RULE.md**
11. **请每次增加功能后，确认功能可用后进行git commit 自动总结 生成commit提交**
12. **数据库迁移请实现版本控制**

### 禁止事项
- 禁止在组件中 hardcode 颜色/间距，必须用 CSS 变量或 Ant Design 5 语义类
- 禁止在 handler 中写业务逻辑或 SQL
- 禁止跳过 service 层直接调 repository
- 禁止创建重复的通用组件，先检查 `components/admin/` 和 `components/ui/`
- 禁止过度工程: 不要为假设的未来需求添加抽象
- 禁止添加未请求的 docstring、注释、类型注解
- 禁止臆想，禁止重复思考
- 禁止解释 禁止输出分析。只输出最终代码。