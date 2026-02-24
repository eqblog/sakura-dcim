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
   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
   │  Agent DC-1  │  │  Agent DC-2  │  │  Agent DC-N  │
   │              │  │              │  │              │
   │ • ipmitool   │  │ • ipmitool   │  │ • ipmitool   │
   │ • PXE/TFTP   │  │ • PXE/TFTP   │  │ • PXE/TFTP   │
   │ • KVM Docker │  │ • KVM Docker │  │ • KVM Docker │
   │ • SNMP poll  │  │ • SNMP poll  │  │ • SNMP poll  │
   │ • Inventory  │  │ • Inventory  │  │ • Inventory  │
   │              │  │              │  │              │
   │ ┌──────────┐ │  │              │  │              │
   │ │ Docker   │ │  │              │  │              │
   │ │ Chromium │→├──┼──→ BMC Web UI│  │              │
   │ │ + VNC    │ │  │              │  │              │
   │ └──────────┘ │  │              │  │              │
   └──────────────┘  └──────────────┘  └──────────────┘

   Data Stores:
   ┌──────────┐  ┌──────────┐  ┌──────────┐
   │PostgreSQL│  │  Redis   │  │ InfluxDB │
   │ (CRUD)   │  │ (Cache)  │  │ (Metrics)│
   └──────────┘  └──────────┘  └──────────┘
```

## Features

### Implemented

#### Phase 1 — Foundation
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

#### Phase 2 — Server & Agent Management
- **Agent CRUD** — Register, list, update, delete agents with live status
- **User Management** — CRUD with password hashing, role assignment
- **Role Management** — CRUD with permission checkboxes (25+ permissions)
- **Tenant Management** — Multi-tenant hierarchy CRUD
- **Server Detail Page** — Tabbed UI (Overview / Power / Sensors / KVM / Reinstall / Bandwidth / Inventory)
- **Agent Detail Page** — Live status, version, capabilities display

#### Phase 3 — IPMI Power Control + Sensor Monitoring
- **Power Control** — On / Off / Reset / Cycle via agent-proxied ipmitool with confirmation dialogs
- **Live Power Status** — Real-time power state polling (15s interval) with status badge
- **IPMI Sensors** — Temperature, fan speed, voltage readings via ipmitool SDR with status tags
- **Backend IPMI Service** — Credential decryption, agent dispatch via WebSocket hub
- **Backend IPMI Handler** — `POST /servers/:id/power` + `GET /servers/:id/power` + `GET /servers/:id/sensors`
- **RBAC Protected** — `server.power` permission for power control, `ipmi.sensors` for sensor reading

#### Phase 4 — NoVNC KVM Console (Docker Browser Isolation)
- **KVM Docker Image** — Alpine + Xvfb + x11vnc + Chromium kiosk mode
- **Agent KVM Executor** — Docker container lifecycle management + VNC TCP↔WebSocket relay
- **Backend KVM Service** — Session management, IPMI credential decryption, agent dispatch
- **Backend KVM Proxy** — Dual WebSocket relay (`/kvm/ws` browser ↔ `/kvm/relay` agent)
- **Frontend noVNC** — One-click KVM console, fullscreen toggle, auto-disconnect on close
- **Universal BMC Support** — Works with iDRAC, iLO, Supermicro, ASRock, any BMC with web UI

#### Phase 5 — PXE OS Reinstall + Auto RAID + Scripts
- **OS Profiles CRUD** — Manage OS templates (Kickstart, Preseed, Autoinstall, cloud-init) with inline editor
- **Disk Layouts CRUD** — JSON-based partition table templates
- **Post-Install Scripts CRUD** — Shell scripts with run order, linked to OS profiles
- **Reinstall Wizard** — 4-step wizard: OS selection → RAID/disk config → credentials/SSH keys → review & confirm
- **Install Task Workflow** — pending → pxe_booting → installing → post_scripts → completed/failed with progress tracking
- **Template Engine** — Go text/template rendering with server variables (IP, hostname, MAC, SSH keys, password hash)
- **Auto RAID Logic** — Automatic RAID level selection based on disk count (1=none, 2=RAID1, 3=RAID5, 4+=RAID10)
- **Agent PXE Executor** — dnsmasq DHCP/TFTP config per server MAC, kernel/initrd download, IPMI PXE boot
- **Agent RAID Executor** — Hardware RAID (storcli) + Software RAID (mdadm) configuration
- **Real-time Progress** — Agent sends pxe.status events → backend updates task + server status, frontend polls 5s

#### Phase 6 — Switch Automation + Bandwidth Monitoring
- **Switch CRUD** — Full management with SSH/SNMP credentials, vendor type (Cisco, Juniper, Arista, SONiC, Cumulus)
- **Switch Port CRUD** — Port mapping with VLAN, speed, admin/oper status, server assignment
- **SSH Port Provisioning** — One-click auto-configure via agent SSH executor with vendor-specific CLI templates
- **Live Port Status** — Query real-time port state via agent SSH/SNMP
- **SNMP Bandwidth Polling** — Agent polls ifInOctets/ifOutOctets via snmpwalk, sends data to backend
- **95th Percentile** — Bandwidth summary with 95th percentile, average, and max calculations
- **Bandwidth Tab** — Per-server bandwidth stats with period selector (hourly/daily/monthly)
- **Agent Switch Executor** — SSH session management (golang.org/x/crypto/ssh) with vendor command templates
- **Agent SNMP Executor** — snmpwalk-based interface counter polling with parsed port traffic data
- **DB Migration** — Version-controlled migration adding SSH creds, vendor, VLAN, status fields

#### Phase 7 — Hardware Inventory + IP Pool Management
- **Inventory Scan** — Agent-triggered hardware detection (lscpu, dmidecode, lsblk, ip addr) stored per-component
- **Inventory Tab** — Rich component display: CPU details, disk table (SSD/HDD/NVMe), network interfaces, memory & system info
- **Inventory Events** — Agent can push inventory.result events for automatic collection
- **IP Pool CRUD** — CIDR-based pools with gateway, usage progress bar (total/used counts)
- **IP Address CRUD** — Per-pool address management with status (available/assigned/reserved) and server assignment
- **Auto-Assign** — One-click next-available IP assignment from pool to server
- **Backend Repository Layer** — InventoryRepository (upsert/list/delete), IPPoolRepository, IPAddressRepository with PostgreSQL
- **Backend Services** — InventoryService (scan via agent, store results), IPService (pool/address CRUD, auto-assign)
- **RBAC Protected** — `inventory.view`, `inventory.scan`, `ip.manage` permissions

#### Phase 8 — White-Label + Multi-Tenant Polish
- **Public Branding API** — `GET /auth/branding` resolves domain/slug → tenant branding (logo, color, favicon)
- **Dynamic Theme** — Ant Design ConfigProvider driven by tenant `primary_color` from Zustand branding store
- **Login Branding** — Login page displays tenant logo, name, and gradient from branding config
- **AppLayout Branding** — Sidebar logo + org name dynamically sourced from branding store
- **Settings Page** — Full white-label config: org name, slug, color picker, logo URL, favicon URL, custom domain
- **Tenant Branding Columns** — Tenants table shows color swatch + logo preview, forms include all branding fields
- **Favicon + Title** — Document title and favicon link updated dynamically on branding load

#### Phase 9 — Audit Hardening + Security + Infrastructure
- **Enhanced Audit Middleware** — Captures sanitized request body (passwords/tokens auto-redacted), extracts resource type + ID from route params
- **Audit Handler** — `GET /audit-logs` with filtering by action, resource_type, user_id, and date range (RBAC: `audit.view`)
- **Rate Limiter** — Redis sorted-set sliding window: 20 req/min on auth endpoints, per-user or per-IP (fail-open on Redis errors)
- **Security Headers** — CSP, X-Frame-Options DENY, X-Content-Type-Options nosniff, X-XSS-Protection, Referrer-Policy, Permissions-Policy, HSTS
- **Audit Log UI** — Advanced filters (action search, resource type dropdown, date range picker), HTTP status tags, detail modal with full JSON
- **InfluxDB Integration** — Time-series storage for bandwidth + sensor data with fail-open fallback to in-memory
- **Prometheus Metrics** — HTTP request counters, duration histograms, custom gauges (agents_online, servers_total, kvm_sessions), `/metrics` endpoint
- **API Documentation Page** — Collapsible endpoint reference with method badges, paths, permissions, descriptions
- **Automated Backups** — PostgreSQL pg_dump cron via Docker container with configurable retention (default 7 days)
- **Agent Config Hot-Reload** — fsnotify file watcher with mutex-protected config reload and callback
- **Agent SOL Support** — IPMI Serial Over LAN (activate/deactivate/info) via ipmitool
- **Agent PXE Inventory Mode** — Boot server to mini-Linux for hardware scanning via dnsmasq + IPMI PXE boot
- **Reseller Hierarchy** — Nested tenant tree with recursive CTE queries, BuildTree algorithm, reseller dashboard
- **IP Assignment Modal** — Server detail IP tab with pool selection, one-click assign, unassign

### Planned

| Feature | Description |
|---------|-------------|
| **IPMI Sensor Graphs** | Temperature, fan speed, voltage time-series charts via InfluxDB (frontend charting) |

## Project Structure

```
sakura-dcim/
├── backend/                    # Go API server
│   ├── cmd/server/main.go      # Entry point
│   ├── internal/
│   │   ├── config/             # Viper configuration
│   │   ├── domain/             # Entity models (8 files)
│   │   ├── handler/            # Gin HTTP handlers
│   │   ├── middleware/         # Auth, RBAC, Audit, Rate-limit, Prometheus
│   │   ├── repository/        # PostgreSQL + InfluxDB implementations
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
│       └── executor/           # IPMI, KVM, PXE, RAID, Inventory executors
│
├── web/                        # React frontend
│   └── src/
│       ├── api/                # Axios client + all API endpoints
│       ├── components/Layout/  # App shell with sidebar
│       ├── pages/              # 12 pages (Dashboard, Servers, Agents, ...)
│       ├── store/              # Zustand stores (auth, branding)
│       └── types/              # Full TypeScript interfaces
│
├── docker/                     # Dockerfiles + nginx config
│   ├── kvm-browser/            # KVM Docker image (Chromium + VNC)
│   └── backup/                 # PostgreSQL backup cron container
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
| `ipmi.kvm.start` | Panel → Agent | Start KVM Docker container + VNC relay |
| `ipmi.kvm.stop` | Panel → Agent | Stop KVM session + destroy container |
| `pxe.prepare` | Panel → Agent | Configure DHCP/TFTP for reinstall |
| `pxe.status` | Agent → Panel | Installation progress updates |
| `raid.configure` | Panel → Agent | Set up RAID before OS install |
| `switch.provision` | Panel → Agent | Auto-configure switch port (VLAN, speed, description) |
| `switch.status` | Panel → Agent | Query switch port status via SSH/SNMP |
| `inventory.scan` | Panel → Agent | Trigger hardware detection |
| `snmp.poll` | Panel → Agent | Poll switch bandwidth counters |

## KVM Architecture (Docker Browser Isolation)

Unlike traditional KVM solutions that require direct VNC/Java access to the BMC, Sakura DCIM uses **Docker browser isolation** for universal BMC compatibility:

```
Browser (noVNC)          Backend (WS relay)         Agent                    Docker Container
     │                        │                       │                          │
     │──POST /servers/:id/kvm─►│                       │                          │
     │                        │──ipmi.kvm.start──────►│                          │
     │                        │                       │──docker run────────────►│
     │                        │                       │  (Chromium kiosk + VNC)  │
     │                        │                       │◄──container ready────────│
     │                        │                       │──TCP VNC connect───────►│
     │                        │◄──{session_id}─────────│                          │
     │◄──{ws_url}─────────────│                       │                          │
     │                        │                       │                          │
     │──WS /kvm/ws──────────►│                       │                          │
     │                        │◄──WS /kvm/relay────────│                          │
     │                        │                       │                          │
     │◄═══VNC binary relay═══►│◄═══VNC binary relay═══►│◄═══TCP VNC═══════════►│──→ BMC Web UI
     │    (noVNC renders)     │    (WebSocket bridge)  │    (raw VNC)            │    (https://IPMI_IP)
```

**Why this approach?**
- **Universal**: Works with any BMC type (iDRAC, iLO, Supermicro, ASRock) — no vendor-specific protocol needed
- **Secure**: Users never access BMC directly; Docker container is network-isolated to only reach the target BMC IP
- **Compatible**: Even Java-based KVM applets work inside the Chromium container
- **Credential-safe**: IPMI passwords are injected into the container, never exposed to the end user

**Per-session resources**: ~200-400MB RAM, 1 CPU core, auto-destroyed on disconnect or 30-min timeout.

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

# 3. Build KVM browser image (required for KVM console)
docker build -t sakura-dcim/kvm-browser:latest docker/kvm-browser/

# 4. Start backend (terminal 1)
make dev-backend
# → API server at http://localhost:8080

# 5. Start frontend (terminal 2)
make dev-web
# → Web UI at http://localhost:5173

# 6. Login
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
  GET    /api/v1/auth/me             # Current user (includes tenant branding)
  POST   /api/v1/auth/logout         # Logout
  GET    /api/v1/auth/branding       # Public tenant branding (by domain/slug) ✅

Servers:
  GET    /api/v1/servers              # List (search, filter, paginate)
  POST   /api/v1/servers              # Create
  GET    /api/v1/servers/:id          # Detail
  PUT    /api/v1/servers/:id          # Update
  DELETE /api/v1/servers/:id          # Delete
  POST   /api/v1/servers/:id/power    # Power control ✅
  GET    /api/v1/servers/:id/power    # Power status ✅
  GET    /api/v1/servers/:id/sensors  # IPMI sensors ✅
  POST   /api/v1/servers/:id/kvm     # Start KVM session ✅
  DELETE /api/v1/servers/:id/kvm     # Stop KVM session ✅
  POST   /api/v1/servers/:id/reinstall         # OS reinstall ✅
  GET    /api/v1/servers/:id/reinstall/status  # Install progress ✅
  GET    /api/v1/servers/:id/bandwidth         # Bandwidth stats ✅
  GET    /api/v1/servers/:id/inventory          # Hardware inventory ✅
  POST   /api/v1/servers/:id/inventory/scan     # Trigger inventory scan ✅

KVM WebSocket:
  WS     /api/v1/kvm/ws              # Browser noVNC connection ✅
  WS     /api/v1/kvm/relay           # Agent VNC relay connection ✅

Agents:
  GET    /api/v1/agents               # List
  POST   /api/v1/agents               # Register
  WS     /api/v1/agents/ws            # Agent WebSocket

Resources:
  CRUD   /api/v1/os-profiles          # OS templates
  CRUD   /api/v1/disk-layouts         # Partition templates
  CRUD   /api/v1/scripts              # Post-install scripts
  CRUD   /api/v1/switches             # Switch management (SSH/Netconf/SNMP) ✅
  CRUD   /api/v1/switches/:id/ports  # Switch port management ✅
  POST   /api/v1/switches/:id/ports/:portId/provision  # Auto-provision port ✅
  GET    /api/v1/switches/:id/ports/:portId/status     # Live port status ✅
  CRUD   /api/v1/ip-pools             # IP address pools ✅
  CRUD   /api/v1/ip-pools/:id/addresses         # IP addresses per pool ✅
  POST   /api/v1/ip-pools/:id/assign            # Auto-assign next available ✅
  CRUD   /api/v1/users                # User management
  CRUD   /api/v1/roles                # Role management
  CRUD   /api/v1/tenants              # Tenant management

Tenants (Reseller Hierarchy):
  GET    /api/v1/tenants/:id/children  # Direct child tenants ✅
  GET    /api/v1/tenants/:id/tree      # Full tenant sub-tree (recursive) ✅

Audit:
  GET    /api/v1/audit-logs           # Search & filter (action, resource_type, user_id, date range) ✅

Monitoring:
  GET    /metrics                     # Prometheus metrics endpoint ✅
```

## Implementation Roadmap

| Phase | Scope | Status |
|-------|-------|--------|
| 1 | Foundation — Auth, RBAC, DB, Layout, Docker | ✅ Done |
| 2 | Server CRUD + Agent WebSocket wiring | ✅ Done |
| 3 | IPMI Power Control + Sensor Monitoring | ✅ Done |
| 4 | NoVNC KVM Console (Docker Browser Isolation) | ✅ Done |
| 5 | PXE OS Reinstall + Auto RAID + Scripts | ✅ Done |
| 6 | Switch Automation + Bandwidth Monitoring | ✅ Done |
| 7 | Hardware Inventory + IP Management | ✅ Done |
| 8 | White-Label + Multi-Tenant Polish | ✅ Done |
| 9 | Audit Hardening + API Docs + Security | ✅ Done |

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
- [x] `agent` Config hot-reload support (fsnotify file watcher with callback)

### Phase 3 — IPMI & Power Management ✅
- [x] `backend` IPMI service: power control + sensors via agent WebSocket (decrypt creds → send to agent)
- [x] `backend` IPMI handler: POST /servers/:id/power, GET /servers/:id/power
- [x] `backend` IPMI sensor handler: GET /servers/:id/sensors
- [x] `web` Server Power tab: buttons (On/Off/Reset/Cycle) + live status badge + confirmation dialogs
- [x] `web` Server Sensors tab: real-time sensor table with status tags + auto-refresh
- [x] `backend` Sensor data collector → write to InfluxDB on interval (InfluxDB bandwidth repo)
- [x] `backend` InfluxDB repository: write sensor readings, query time-series (influxdb-client-go/v2)
- [x] `agent` IPMI executor: SOL (Serial Over LAN) support (activate/deactivate/info via ipmitool)

### Phase 4 — NoVNC KVM Console (Docker Browser Isolation) ✅
- [x] `docker` KVM browser image: Alpine + Xvfb + x11vnc + Chromium kiosk mode
- [x] `agent` KVM executor: Docker container lifecycle + VNC TCP↔WebSocket relay
- [x] `backend` KVM service: session management, IPMI credential decryption, agent dispatch
- [x] `backend` KVM handler: POST /servers/:id/kvm + WebSocket relay (browser↔backend↔agent)
- [x] `backend` KVM WebSocket proxy: /kvm/ws (browser) + /kvm/relay (agent) dual-endpoint
- [x] `web` KVM Console page: noVNC integration, fullscreen toggle, auto-disconnect on close

### Phase 5 — PXE & OS Reinstallation ✅
- [x] `backend` OS Profile CRUD handler + service
- [x] `backend` Disk Layout CRUD handler + service
- [x] `backend` Script CRUD handler + service
- [x] `backend` Install task workflow: create → pxe_booting → installing → post_scripts → completed/failed
- [x] `backend` Template engine: render Kickstart/Preseed/Autoinstall/cloud-init with server variables
- [x] `backend` Auto RAID logic: pick RAID level based on disk count (1 disk=none, 2=RAID1, 3+=RAID5, 4+=RAID10)
- [x] `agent` PXE executor: manage dnsmasq DHCP/TFTP config per server MAC
- [x] `agent` RAID executor: storcli/megacli (HW RAID) + mdadm (SW RAID)
- [x] `agent` Post-install callback: report progress via WebSocket event
- [x] `web` OS Profiles page: CRUD with template editor
- [x] `web` Disk Layouts page: CRUD with JSON partition editor
- [x] `web` Scripts page: CRUD with shell editor
- [x] `web` Reinstall Wizard: Select OS → RAID config → Disk layout → SSH keys → Review → Install
- [x] `web` Install progress: real-time progress bar + log stream

### Phase 6 — Switch Automation & Bandwidth Monitoring ✅
- [x] `backend` Switch CRUD handler + service (SSH/Netconf credentials, vendor type)
- [x] `backend` Switch port mapping handler (assign port → server)
- [x] `backend` Switch port provisioning service: auto-configure VLAN, speed, description via agent SSH
- [x] `backend` Bandwidth service: 95th percentile, avg, max calculation with period filtering
- [x] `backend` Bandwidth handler: GET /servers/:id/bandwidth with period parameter
- [x] `backend` DB migration: 000002_switch_automation (SSH creds, vendor, VLAN, status fields)
- [x] `agent` Switch executor: SSH session management (golang.org/x/crypto/ssh) + vendor-specific CLI templates (Cisco, Juniper, Arista, SONiC, Cumulus)
- [x] `agent` SNMP executor: snmpwalk-based interface counter polling (ifDescr/ifInOctets/ifOutOctets/ifSpeed/ifOperStatus)
- [x] `web` Switches page: CRUD with SSH/SNMP credentials + port management table + provision button
- [x] `web` Server Bandwidth tab: per-port stats with 95th percentile, period selector
- [x] `web` Bandwidth overview page: switch list with SNMP status

### Phase 7 — Inventory & IP Management ✅
- [x] `backend` Inventory handler: GET /servers/:id/inventory, POST /servers/:id/inventory/scan
- [x] `backend` Inventory service: store scan results per-component (upsert), handle agent events
- [x] `backend` Inventory repository: PostgreSQL upsert/list/delete for server_inventory table
- [x] `backend` IP Pool CRUD handler + service (with usage counts)
- [x] `backend` IP Address CRUD handler + service (status: available/assigned/reserved)
- [x] `backend` IP Address auto-assign logic (next available from pool)
- [x] `backend` IP Pool/Address repositories: full CRUD + GetNextAvailable
- [x] `agent` Inventory scanner: lscpu, dmidecode, lsblk -J, ip -j addr show (already existed)
- [x] `web` Server Inventory tab: CPU details, disk table, network interfaces, memory/system raw output
- [x] `web` IP Pools page: pool CRUD + usage progress bar + address management panel
- [x] `web` IP Address CRUD: add/edit/delete with status filter
- [x] `agent` PXE inventory mode: boot to mini-Linux, scan, report, reboot (dnsmasq + IPMI PXE boot)
- [x] `web` IP assignment modal in server detail (pool selection, assign/unassign)

### Phase 8 — White-Label & Multi-Tenant Polish ✅
- [x] `backend` Tenant branding API: GET /auth/branding (public, returns logo/color/favicon by domain or slug)
- [x] `backend` AuthService: populate tenant branding on login and /auth/me
- [x] `backend` Custom domain resolution: branding endpoint resolves Host header → tenant
- [x] `backend` Tenant CRUD: already includes logo_url, primary_color, favicon_url, custom_domain fields
- [x] `web` Branding store (Zustand): fetchBranding on app load, dynamic favicon + document title
- [x] `web` Dynamic theme: ConfigProvider uses tenant primary_color from branding store
- [x] `web` AppLayout: dynamic logo (image or icon) + dynamic org name from branding
- [x] `web` Login page: dynamic branding (logo, name, gradient color)
- [x] `web` Settings page: full branding form (name, slug, color picker, logo URL, favicon URL, custom domain)
- [x] `web` Tenants page: enhanced with logo_url, favicon_url fields + branding preview column
- [x] `backend` Reseller hierarchy: nested tenant tree (recursive CTE), cascading permissions
- [x] `web` Reseller dashboard: manage sub-tenants, assign servers (tree view + server table)

### Phase 9 — Audit, Logging & Hardening ✅
- [x] `backend` Audit middleware: capture sanitized request body (passwords/tokens redacted), extract resource type/ID from routes
- [x] `backend` Audit handler: GET /audit-logs with filtering by action, resource_type, user_id, date range
- [x] `backend` Rate limiter: Redis sliding window (20 req/min on auth endpoints, per-user or per-IP)
- [x] `backend` Security headers: CSP, X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, Referrer-Policy, Permissions-Policy, HSTS
- [x] `web` Audit log page: advanced filters (action search, resource type dropdown, date range picker), detail modal, status tags
- [x] `backend` API documentation: comprehensive endpoint reference page
- [x] `web` API documentation page: collapsible endpoint groups with method, path, permission, description
- [x] `infra` Automated backups: PostgreSQL pg_dump cron (Docker container with retention)
- [x] `infra` Prometheus metrics exporter: HTTP counters, duration histograms, custom gauges (/metrics endpoint)

## Security

- **JWT** — Short-lived access tokens (15min) + refresh tokens (7d, HttpOnly cookie)
- **RBAC** — 25+ permissions, middleware-enforced, tenant-scoped
- **Tenant Isolation** — Every query filtered by tenant_id
- **IPMI Credentials** — AES-256-GCM encrypted at rest, decrypted only when sent to agent
- **Agent Auth** — Per-agent tokens, bcrypt hashed, validated on WebSocket connect
- **KVM Sessions** — Short-lived tokens, Docker browser isolation (user never touches BMC directly), 30-min auto-expiry, container destroyed on disconnect
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
make build-kvm       # Build KVM browser Docker image
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