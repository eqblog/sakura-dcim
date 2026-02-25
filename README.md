# Sakura DCIM

Data Center Infrastructure Management — manage dedicated servers across multiple datacenters from one web interface.

## Prerequisites

On a fresh Linux server, the startup scripts auto-install everything. For local dev on macOS/Windows you need:

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)

## One-Click Start

```bash
git clone https://github.com/sakura-dcim/sakura-dcim.git
cd sakura-dcim
```

### Development

| Linux / macOS | Windows |
|---------------|---------|
| `make start` | `scripts\start-dev.bat` |

The dev script (`scripts/start-dev.sh`) does everything automatically:

1. Installs missing dependencies (Docker, Go, Node.js, ipmitool, dmidecode, dnsmasq, etc.)
2. Starts PostgreSQL, Redis, InfluxDB via Docker Compose
3. Runs database migrations
4. Installs frontend npm packages
5. Starts backend API server
6. Builds KVM browser Docker image
7. Creates and starts a local dev agent (auto-registers via API)
8. Starts frontend dev server with API proxy

```bash
# Default: frontend 0.0.0.0:5173, backend :8080
bash scripts/start-dev.sh

# Custom ports (e.g. for public access):
FRONTEND_PORT=3000 BACKEND_PORT=9090 bash scripts/start-dev.sh

# Custom frontend bind address:
FRONTEND_HOST=0.0.0.0 FRONTEND_PORT=80 bash scripts/start-dev.sh
```

| Variable | Default | Description |
|----------|---------|-------------|
| `FRONTEND_HOST` | `0.0.0.0` | Frontend bind address |
| `FRONTEND_PORT` | `5173` | Frontend port |
| `BACKEND_PORT` | `8080` | Backend API port |

- Frontend: http://localhost:5173
- Backend: http://localhost:8080
- Login: `admin@sakura-dcim.local` / `admin123`

### Production

| Linux / macOS | Windows |
|---------------|---------|
| `make start-prod` | `scripts\start-prod.bat` |

The prod script (`scripts/start-prod.sh`) handles:

1. Installs Docker & Docker Compose if missing
2. Builds and starts all services via `docker compose up -d --build`
3. Waits for PostgreSQL, runs database migrations inside the container
4. Optionally deploys a local agent (`WITH_AGENT=1`)

```bash
# Default: web UI on port 3000
bash scripts/start-prod.sh

# Custom web port:
WEB_PORT=80 bash scripts/start-prod.sh

# With local agent (auto-installs Go, ipmitool, dmidecode, KVM image):
WITH_AGENT=1 bash scripts/start-prod.sh

# Full custom:
WEB_PORT=443 API_PORT=9090 WITH_AGENT=1 bash scripts/start-prod.sh
```

| Variable | Default | Description |
|----------|---------|-------------|
| `WEB_PORT` | `3000` | Web UI port |
| `API_PORT` | `8080` | Backend API port |
| `WITH_AGENT` | `0` | Set to `1` to deploy a local agent |

Web UI at http://localhost:3000.

### Auto-Installed Tools

When running with an agent (dev mode or `WITH_AGENT=1`), the scripts auto-install:

| Tool | Purpose |
|------|---------|
| `ipmitool` | IPMI power control & sensor reading |
| `dmidecode` | Hardware inventory (system/BIOS/memory info) |
| `dnsmasq` | PXE boot DHCP/TFTP server |
| `snmpwalk` | Switch bandwidth polling |
| `mdadm` | Software RAID configuration |
| `lscpu`, `lsblk`, `ip` | Hardware inventory scanning |
| `pciutils` | PCI device detection |

Supports apt (Debian/Ubuntu), dnf (Fedora/RHEL 8+), yum (CentOS/RHEL 7), and apk (Alpine).

## One-Click Update

| Linux / macOS | Windows |
|---------------|---------|
| `make update` | `scripts\update.bat` |

Automatically: `git pull` → update Go & npm deps → run new migrations.

Then restart with the start command above.

## Stop

```bash
make stop
# or
docker compose down
```

## Deploy Agent

Each datacenter needs one agent. In dev mode, a local agent is created automatically.

For remote datacenters:

```bash
# 1. Web panel → Agents → Add Agent → copy ID & Token
# 2. On datacenter server:
cd agent
cp config.yaml.example config.yaml
# Edit config.yaml with panel URL, agent ID, token
go run ./cmd/agent
```

The agent requires these tools on the host: `ipmitool`, `dmidecode`, `dnsmasq`, `docker`.

## Configuration

The backend uses environment variables (prefix `SAKURA_`) or `backend/config.yaml`. Defaults work for local dev.

Production — set these environment variables:

```bash
SAKURA_JWT_SECRET=your-jwt-secret-at-least-32-chars
SAKURA_CRYPTO_ENCRYPTION_KEY=64-hex-chars-for-aes256gcm
SAKURA_DATABASE_PASSWORD=strong-db-password
SAKURA_SERVER_MODE=release
```

<details>
<summary>All environment variables</summary>

| Variable | Default | Description |
|----------|---------|-------------|
| `SAKURA_DATABASE_HOST` | `localhost` | PostgreSQL host |
| `SAKURA_DATABASE_PORT` | `5432` | PostgreSQL port |
| `SAKURA_DATABASE_USER` | `sakura` | PostgreSQL user |
| `SAKURA_DATABASE_PASSWORD` | `sakura` | PostgreSQL password |
| `SAKURA_DATABASE_DBNAME` | `sakura_dcim` | Database name |
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
| `SAKURA_CRYPTO_ENCRYPTION_KEY` | *(required)* | 64 hex chars for AES-256-GCM |
| `SAKURA_SERVER_HOST` | `0.0.0.0` | Bind address |
| `SAKURA_SERVER_PORT` | `8080` | Listen port |
| `SAKURA_SERVER_MODE` | `debug` | `debug` / `release` / `test` |

</details>

## Tests

```bash
make test          # Run all backend tests
make test-verbose  # Verbose output
make lint          # Run linter
```

## All Make Commands

| Command | Description |
|---------|-------------|
| `make start` | **One-click dev start** (infra + migrate + backend + frontend) |
| `make start-prod` | **One-click production start** (docker compose) |
| `make update` | **One-click update** (git pull + deps + migrate) |
| `make stop` | Stop all services |
| `make test` | Run backend tests |
| `make migrate` | Apply database migrations |
| `make migrate-down` | Rollback last migration |
| `make clean` | Remove build artifacts |

<details>
<summary>More commands</summary>

| Command | Description |
|---------|-------------|
| `make dev-backend` | Run backend only |
| `make dev-web` | Run frontend only |
| `make build-backend` | Build backend binary |
| `make build-agent` | Build agent binary |
| `make build-web` | Build frontend |
| `make docker-infra` | Start PG + Redis + InfluxDB |
| `make docker-up` | Start all Docker services |
| `make docker-down` | Stop all Docker services |
| `make docker-build` | Build Docker images |
| `make lint` | Run Go linter |
| `make migrate-reset` | Reset database (drops all data) |

</details>

## Project Structure

```
sakura-dcim/
├── backend/          # Go API server (Gin)
├── agent/            # Go agent (per datacenter)
├── web/              # React frontend (Ant Design)
├── docker/           # Dockerfiles + nginx
├── scripts/          # One-click start/update scripts
├── docker-compose.yml
├── Makefile
└── README_AI.md      # Detailed implementation notes
```

## License

Private — All rights reserved.
