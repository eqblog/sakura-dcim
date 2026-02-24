# Sakura DCIM

Data Center Infrastructure Management — manage dedicated servers across multiple datacenters from one web interface.

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)

## One-Click Start

### Development

```bash
git clone https://github.com/sakura-dcim/sakura-dcim.git
cd sakura-dcim
make start
```

This will automatically:
1. Start PostgreSQL + Redis + InfluxDB (Docker)
2. Run database migrations
3. Install frontend dependencies
4. Start backend (http://localhost:8080) and frontend (http://localhost:5173)

Login: `admin@sakura-dcim.local` / `admin123`

### Production

```bash
make start-prod
```

Builds and starts all services via Docker Compose. Web UI at http://localhost:3000.

## One-Click Update

```bash
make update
```

This will automatically:
1. `git pull` latest code
2. Update Go and npm dependencies
3. Run new database migrations

Then restart:

```bash
# Development
make start

# Production
make start-prod
```

## Stop

```bash
make stop
```

## Deploy Agent

Each datacenter needs one agent:

```bash
# 1. Web panel → Agents → Add Agent → copy ID & Token
# 2. On datacenter server:
cd agent
cp config.yaml.example config.yaml
# Edit config.yaml with panel URL, agent ID, token
go run ./cmd/agent
```

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
