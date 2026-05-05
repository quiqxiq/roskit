BINARY_DIR := bin
API_BIN     := $(BINARY_DIR)/api
MIGRATE_BIN := $(BINARY_DIR)/migrate
WORKER_BIN  := $(BINARY_DIR)/worker
SEED_BIN    := $(BINARY_DIR)/seed

COMPOSE     := docker compose -f docker/docker-compose.dev.yml
COMPOSE_PRD := docker compose -f docker/docker-compose.yml

API_URL     := http://localhost:8080

.PHONY: all build build-api build-migrate build-worker build-seed \
        run docker-up docker-down docker-logs docker-build docker-clean \
        migrate-up migrate-import sync-profiles seed seed-docker \
        test test-mikrotik test-redis test-influxdb test-integration \
        clean env curl-health curl-setup curl-login

# ── Build ────────────────────────────────────────────────────────────────────

all: build

build: build-api build-migrate build-worker

build-api:
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(API_BIN) ./cmd/api

build-migrate:
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(MIGRATE_BIN) ./cmd/migrate

build-worker:
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(WORKER_BIN) ./cmd/worker

build-seed:
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(SEED_BIN) ./cmd/seed

clean:
	rm -rf $(BINARY_DIR)

# ── Local dev ────────────────────────────────────────────────────────────────

env:
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example"; else echo ".env already exists"; fi

run: build-api
	./$(API_BIN)

# ── Docker ───────────────────────────────────────────────────────────────────

docker-build:
	$(COMPOSE) build

docker-up:
	$(COMPOSE) up -d --build
	@echo "Stack is up. Run 'make migrate-up-docker' to apply schema."

docker-down:
	$(COMPOSE) down

docker-logs:
	$(COMPOSE) logs -f api

docker-clean:
	$(COMPOSE) down -v --remove-orphans

# Run migrations inside the running api container
migrate-up-docker:
	$(COMPOSE) exec api /migrate up

# ── Migrations (local binary) ─────────────────────────────────────────────────

migrate-up: build-migrate
	./$(MIGRATE_BIN) up

migrate-import: build-migrate
	@if [ -z "$(CONFIG_FILE)" ]; then echo "Usage: make migrate-import CONFIG_FILE=/path/to/config.php"; exit 1; fi
	./$(MIGRATE_BIN) import --config-file=$(CONFIG_FILE)

sync-profiles: build-migrate
	./$(MIGRATE_BIN) sync-profiles

seed: build-seed
	./$(SEED_BIN)

seed-docker:
	$(COMPOSE) exec api /seed

# ── Testing ───────────────────────────────────────────────────────────────────

test:
	go test -race ./...

test-mikrotik:
	@echo "Running MikroTik integration tests..."
	set -a && source .env.test && set +a && go test -race -tags mikrotik -count=1 -v ./internal/roskit/execution/ ./internal/roskit/orchestrator/ ./internal/roskit/adapter/service/

test-redis:
	@echo "Running Redis integration tests..."
	set -a && source .env.test && set +a && go test -race -tags redis -count=1 -v ./internal/roskit/pipeline/cache/ ./internal/roskit/pipeline/pubsub/ ./internal/roskit/pipeline/event/

test-influxdb:
	@echo "Running InfluxDB integration tests..."
	set -a && source .env.test && set +a && go test -race -tags influxdb -count=1 -v ./internal/roskit/pipeline/timeseries/

test-integration:
	@echo "Running all integration tests..."
	set -a && source .env.test && set +a && go test -race -tags 'mikrotik redis influxdb' -count=1 -v ./...

# ── Curl helpers ─────────────────────────────────────────────────────────────

curl-health:
	@curl -s $(API_URL)/api/v1/events/health | python3 -m json.tool 2>/dev/null || curl -s $(API_URL)/api/v1/events/health

curl-setup:
	@curl -s -X POST $(API_URL)/api/v1/auth/setup \
		-H "Content-Type: application/json" \
		-d '{"username":"admin","password":"admin1234"}' \
		| python3 -m json.tool 2>/dev/null || true

curl-login:
	@curl -s -X POST $(API_URL)/api/v1/auth/login \
		-H "Content-Type: application/json" \
		-d '{"username":"admin","password":"admin1234"}' \
		| python3 -m json.tool 2>/dev/null || true
