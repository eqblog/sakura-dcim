.PHONY: dev dev-backend dev-web build migrate migrate-down docker-up docker-down clean

# Development
dev: docker-infra dev-backend dev-web

dev-backend:
	cd backend && go run ./cmd/server

dev-web:
	cd web && npm run dev

# Build
build-backend:
	cd backend && CGO_ENABLED=0 go build -o bin/sakura-server ./cmd/server

build-agent:
	cd agent && CGO_ENABLED=0 go build -o bin/sakura-agent ./cmd/agent

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

# Clean
clean:
	rm -rf backend/bin agent/bin web/dist web/node_modules

# Test
test-backend:
	cd backend && go test ./...

# Lint
lint-backend:
	cd backend && golangci-lint run ./...
