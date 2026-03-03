.PHONY: start start-prod update stop dev-backend dev-web build migrate migrate-down docker-up docker-down clean test deploy-agent build-kvm

# ============================================================
# One-click commands
# ============================================================

## Start development environment (infra + migrate + backend + frontend)
start:
	bash scripts/start-dev.sh

## Start production environment (docker compose)
start-prod:
	bash scripts/start-prod.sh

## Pull latest code + update deps + migrate
update:
	bash scripts/update.sh

## Stop all services
stop:
	docker compose down

# ============================================================
# Individual commands
# ============================================================

# Development
dev-backend:
	cd backend && go run ./cmd/server

dev-web:
	cd web && npm run dev

# Build
build-backend:
	cd backend && CGO_ENABLED=0 go build -o bin/sakura-server ./cmd/server

build-agent:
	cd agent && CGO_ENABLED=0 go build -o bin/sakura-agent ./cmd/agent

build-agent-linux:
	cd agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/sakura-agent-linux-amd64 ./cmd/agent

deploy-agent:
	bash scripts/deploy-agent.sh

build-web:
	cd web && npm run build

# Database migrations
migrate:
	cd backend && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" up

migrate-down:
	cd backend && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" down 1

migrate-reset:
	cd backend && go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path migrations -database "postgres://sakura:sakura@localhost:5432/sakura_dcim?sslmode=disable" drop -f

# Docker
docker-infra:
	docker compose up -d postgres redis influxdb

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

build-kvm:
	docker build -t sakura-dcim/kvm-browser:latest docker/kvm-browser/

# Test
test:
	cd backend && go test ./...

test-verbose:
	cd backend && go test -v ./...

# Lint
lint:
	cd backend && golangci-lint run ./...

# Clean
clean:
	rm -rf backend/bin agent/bin web/dist web/node_modules
