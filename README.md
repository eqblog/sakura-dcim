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

## Prerequisites

- **Go** 1.25+
- **Node.js** 20+ (with npm)
- **Docker** & **Docker Compose**
- **Make** (optional, for convenience commands)

## Quick Start (Development)

### 1. Clone the repository

```bash
git clone https://github.com/sakura-dcim/sakura-dcim.git
cd sakura-dcim
```

### 2. Start infrastructure services

Start PostgreSQL, Redis, and InfluxDB via Docker:

```bash
make docker-infra
# or manually:
docker compose up -d postgres redis influxdb
```

### 3. Run database migrations

```bash
make migrate
```

This applies all SQL migrations from `backend/migrations/` to the PostgreSQL database.

### 4. Configure the backend (optional)

The backend reads configuration from environment variables (prefix `SAKURA_`) or a `config.yaml` file. Defaults are provided for local development:

| Variable | Default | Description |
|----------|---------|-------------|
| `SAKURA_DATABASE_HOST` | `localhost` | PostgreSQL host |
| `SAKURA_DATABASE_PORT` | `5432` | PostgreSQL port |
| `SAKURA_DATABASE_USER` | `sakura` | PostgreSQL user |
| `SAKURA_DATABASE_PASSWORD` | `sakura` | PostgreSQL password |
| `SAKURA_DATABASE_DBNAME` | `sakura_dcim` | PostgreSQL database name |
| `SAKURA_DATABASE_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `SAKURA_REDIS_HOST` | `localhost` | Redis host |
| `SAKURA_REDIS_PORT` | `6379` | Redis port |
| `SAKURA_INFLUXDB_URL` | `http://localhost:8086` | InfluxDB URL |
| `SAKURA_INFLUXDB_TOKEN` | *(empty)* | InfluxDB auth token |
| `SAKURA_INFLUXDB_ORG` | `sakura` | InfluxDB organization |
| `SAKURA_INFLUXDB_BUCKET` | `dcim` | InfluxDB bucket |
| `SAKURA_JWT_SECRET` | *(required)* | JWT signing secret (32+ chars) |
| `SAKURA_JWT_ACCESS_TOKEN_TTL` | `15` | Access token lifetime (minutes) |
| `SAKURA_JWT_REFRESH_TOKEN_TTL` | `168` | Refresh token lifetime (hours) |
| `SAKURA_CRYPTO_ENCRYPTION_KEY` | *(required)* | 64 hex chars (32 bytes) for AES-256-GCM |
| `SAKURA_SERVER_HOST` | `0.0.0.0` | Server bind address |
| `SAKURA_SERVER_PORT` | `8080` | Server listen port |
| `SAKURA_SERVER_MODE` | `debug` | Gin mode: `debug`, `release`, `test` |

You can also place a `backend/config.yaml`:

```yaml
jwt:
  secret: "your-jwt-secret-at-least-32-characters"
  access_token_ttl: 15
  refresh_token_ttl: 168
crypto:
  encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
```

### 5. Build the KVM browser image

Required for KVM console functionality:

```bash
docker build -t sakura-dcim/kvm-browser:latest docker/kvm-browser/
```

### 6. Start the backend

```bash
make dev-backend
# → API server at http://localhost:8080
```

### 7. Install frontend dependencies and start

```bash
cd web && npm install && cd ..
make dev-web
# → Web UI at http://localhost:5173
```

### 8. Login

Open http://localhost:5173 and login with the seed credentials:

- **Email**: `admin@sakura-dcim.local`
- **Password**: `admin123`

## Production Deployment (Docker Compose)

### Full stack deployment

```bash
# Build and start all services (PostgreSQL, Redis, InfluxDB, backend, web, backup)
docker compose up -d --build
```

| Service | URL |
|---------|-----|
| Web UI | http://localhost:3000 |
| API | http://localhost:3000/api/v1 |
| InfluxDB UI | http://localhost:8086 |
| Prometheus Metrics | http://localhost:8080/metrics |

### Production environment variables

Set these in your `docker-compose.yml` or via `.env` file:

```bash
# Required — change these!
SAKURA_JWT_SECRET=your-production-jwt-secret-at-least-32-chars
SAKURA_CRYPTO_ENCRYPTION_KEY=your-64-hex-char-encryption-key-for-aes256gcm

# Database
SAKURA_DATABASE_HOST=postgres
SAKURA_DATABASE_PASSWORD=strong-password-here

# InfluxDB
SAKURA_INFLUXDB_URL=http://influxdb:8086
SAKURA_INFLUXDB_TOKEN=your-influxdb-admin-token

# Server
SAKURA_SERVER_MODE=release
```

## Deploying an Agent

Each datacenter needs one agent running to manage servers:

```bash
# 1. In the web panel: Agents → Add Agent
#    Copy the generated Agent ID and Token

# 2. On the datacenter server:
cd agent
cp config.yaml.example config.yaml
```

Edit `config.yaml`:

```yaml
server_url: "ws://your-panel-domain.com/api/v1/agents/ws"
agent_id: "the-uuid-from-step-1"
token: "the-token-from-step-1"
```

```bash
# 3. Run the agent
go run ./cmd/agent

# Or build and deploy as a systemd service:
CGO_ENABLED=0 go build -o sakura-agent ./cmd/agent
```

## Updating the Project

### Pulling updates

```bash
# 1. Pull latest code
git pull origin main

# 2. Update backend dependencies
cd backend && go mod tidy && cd ..

# 3. Update frontend dependencies
cd web && npm install && cd ..

# 4. Run new database migrations (if any)
make migrate

# 5. Rebuild and restart
# Development:
make dev-backend  # terminal 1
make dev-web      # terminal 2

# Production:
docker compose up -d --build
```

### Database migrations

Migrations are version-controlled in `backend/migrations/`. They run sequentially in order:

```bash
# Apply all pending migrations
make migrate

# Rollback last migration
make migrate-down

# Reset database (DANGER: drops all data)
make migrate-reset
```

### Updating agents

Agents should be updated after the backend is updated:

```bash
cd agent
go build -o sakura-agent ./cmd/agent
# Restart the agent process / systemd service
```

The agent supports config hot-reload — changes to `config.yaml` are picked up automatically without restart.

## Running Tests

```bash
# Backend unit tests
make test-backend
# or: cd backend && go test ./...

# Run specific test (e.g., the full lifecycle integration test)
cd backend && go test -v -run TestServerLifecycle ./internal/handler/

# Backend linting
make lint-backend
```

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
make docker-build    # Build Docker images
make test-backend    # Run Go tests
make lint-backend    # Run Go linter
make clean           # Remove build artifacts
```

## Project Structure

```
sakura-dcim/
├── backend/                    # Go API server
│   ├── cmd/server/main.go      # Entry point
│   ├── internal/
│   │   ├── config/             # Viper configuration
│   │   ├── domain/             # Entity models
│   │   ├── handler/            # Gin HTTP handlers
│   │   ├── middleware/         # Auth, RBAC, Audit, Rate-limit, Prometheus
│   │   ├── repository/        # PostgreSQL + InfluxDB implementations
│   │   ├── service/           # Business logic
│   │   ├── websocket/         # Agent WS hub + protocol
│   │   └── pkg/crypto/        # AES-256-GCM, JWT, bcrypt
│   └── migrations/            # SQL schema migrations
│
├── agent/                      # Lightweight Go agent (per datacenter)
│   ├── cmd/agent/main.go       # Entry point
│   └── internal/
│       ├── client/             # WebSocket client + reconnect
│       ├── config/             # YAML config + hot-reload
│       └── executor/           # IPMI, KVM, PXE, RAID, SNMP, Inventory
│
├── web/                        # React frontend
│   └── src/
│       ├── api/                # Axios client + all API endpoints
│       ├── components/Layout/  # App shell with sidebar
│       ├── pages/              # Dashboard, Servers, Agents, etc.
│       ├── store/              # Zustand stores (auth, branding)
│       └── types/              # TypeScript interfaces
│
├── docker/                     # Dockerfiles + nginx config
│   ├── kvm-browser/            # KVM Docker image (Chromium + VNC)
│   └── backup/                 # PostgreSQL backup cron container
├── docker-compose.yml          # Full stack environment
├── Makefile                    # Dev, build, migrate, test commands
└── README_AI.md                # Detailed implementation notes
```

## Security

- **JWT** — Short-lived access tokens (15min) + refresh tokens (7d)
- **RBAC** — 25+ permissions, middleware-enforced, tenant-scoped
- **Tenant Isolation** — Every query filtered by tenant_id
- **IPMI Credentials** — AES-256-GCM encrypted at rest, decrypted only when sent to agent
- **Agent Auth** — Per-agent tokens, bcrypt hashed, validated on WebSocket connect
- **KVM Sessions** — Docker browser isolation, 30-min auto-expiry
- **Rate Limiting** — Redis sliding window on auth endpoints (20 req/min)
- **Security Headers** — CSP, HSTS, X-Frame-Options, X-Content-Type-Options
- **Audit Trail** — Every mutation logged with user, IP, user-agent, details

## License

Private — All rights reserved.
