# Pyramid — top-level developer tasks.
# openapi/openapi.yaml is the single source of truth for types on both sides.

OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
SPEC := openapi/openapi.yaml
COMPOSE := docker compose

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Code generation — regenerate types on BOTH sides from the OpenAPI contract.
# ---------------------------------------------------------------------------
.PHONY: generate generate-backend generate-frontend
generate: generate-backend generate-frontend ## Regenerate Go + TS types from the OpenAPI spec

generate-backend: ## Regenerate Go DTOs (backend/internal/api/types.gen.go)
	cd backend && $(OAPI_CODEGEN) -config oapi-codegen.yaml -o internal/api/types.gen.go ../$(SPEC)

generate-frontend: ## Regenerate TS types (frontend/src/lib/api/schema.d.ts)
	cd frontend && npx openapi-typescript ../$(SPEC) -o src/lib/api/schema.d.ts

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------
.PHONY: setup setup-backend setup-frontend
setup: setup-backend setup-frontend ## Install all dependencies

setup-backend: ## Download Go modules
	cd backend && go mod tidy

setup-frontend: ## Install npm dependencies
	cd frontend && npm install

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------
.PHONY: db-create db-schema
db-create: ## Create the pyramid database (needs a running Postgres)
	createdb pyramid || true

db-schema: ## Apply db/schema.sql to the pyramid database
	psql -d pyramid -f db/schema.sql

# ---------------------------------------------------------------------------
# Dev servers
# ---------------------------------------------------------------------------
.PHONY: dev-backend dev-frontend
dev-backend: ## Run the Go API (localhost:8080)
	cd backend && go run ./cmd/server

dev-frontend: ## Run the Svelte dev server (localhost:5173)
	cd frontend && npm run dev

# ---------------------------------------------------------------------------
# Build & check
# ---------------------------------------------------------------------------
.PHONY: build build-backend build-frontend check
build: build-backend build-frontend ## Build both apps

build-backend: ## Compile the Go server to backend/bin/server
	cd backend && go build -o bin/server ./cmd/server

build-frontend: ## Production build of the Svelte app
	cd frontend && npm run build

check: ## Type-check the frontend
	cd frontend && npm run check

# ---------------------------------------------------------------------------
# Docker — run the whole stack (Postgres + backend + frontend) in containers
# ---------------------------------------------------------------------------
.PHONY: docker-up docker-down docker-build docker-logs docker-ps docker-restart docker-clean
docker-up: ## Build (if needed) and start all services in the background
	$(COMPOSE) up -d --build
	@echo "Frontend → http://localhost:3000   API → http://localhost:8080   DB → localhost:5433"

docker-build: ## Build the backend + frontend images
	$(COMPOSE) build

docker-down: ## Stop and remove the containers (keeps the database volume)
	$(COMPOSE) down

docker-clean: ## Stop containers AND delete the database volume (fresh schema next up)
	$(COMPOSE) down -v

docker-logs: ## Follow logs from all services
	$(COMPOSE) logs -f

docker-ps: ## Show container status
	$(COMPOSE) ps

docker-restart: ## Rebuild and restart all services
	$(COMPOSE) up -d --build
