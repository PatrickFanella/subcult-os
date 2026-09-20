SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

PROJECT_NAME := subcult-os
COMPOSE_PROJECT_NAME ?= $(PROJECT_NAME)
BACKEND_BIN ?= bin/$(PROJECT_NAME)

.PHONY: help deps deps-web deps-mobile verify quick fmt lint lint-mobile test test-backend test-db test-web test-mobile build build-backend build-web run-backend dev dev-mobile up up-build down reset-db restart logs ps urls smoke alpha-qa alpha-qa-paid fake-event-qa compose-config db-shell migrate migrate-status migrate-reset clean open-pilot-check check-contracts

help:
	@awk 'BEGIN {FS = ":.*##"; printf "$(PROJECT_NAME) commands:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: deps-web deps-mobile ## Install web and mobile dependencies

deps-web: ## Install web dependencies
	pnpm --dir web install --frozen-lockfile

deps-mobile: ## Install mobile dependencies
	pnpm --dir mobile install --frozen-lockfile

verify: deps fmt lint check-contracts test build compose-config open-pilot-check ## Run all checks

quick: fmt lint test ## Run fast local checks without Docker config or production builds

fmt: ## Format/check Go and TypeScript sources
	@git ls-files --cached --others --exclude-standard 'backend/**/*.go' 'backend/*.go' | xargs -r gofmt -w
	pnpm --dir web run format

lint: ## Run Go vet and frontend lint
	cd backend && go vet ./...
	pnpm --dir web run lint
	pnpm --dir mobile run lint

lint-mobile: ## Type-check the Expo mobile app
	pnpm --dir mobile run lint

test: test-backend test-web test-mobile ## Run backend, frontend, and mobile tests

test-backend: ## Run non-DB Go tests
	cd backend && TEST_DATABASE_URL= go test ./...

test-db: ## Run DB-backed Go integration tests when TEST_DATABASE_URL is set
	@if [ -z "$${TEST_DATABASE_URL}" ]; then \
		echo "TEST_DATABASE_URL is required for DB-backed tests"; \
		exit 1; \
	fi
	cd backend && go test ./internal/app -run 'TestFirstEventLifecycleCurrentCreatePublishFreeDoorEndOfNightFlow|TestTicketReservationCurrentCapacityAndDoorRules|TestRunMigrations|TestIdentity|TestRotatedRefresh|TestSessionExpiry|TestPasswordRecovery|TestPrototypeSession' -count=1 -v
	cd backend && go test ./internal/atproto -run 'TestOAuthStore' -count=1 -v

test-web: ## Run frontend tests
	pnpm --dir web run test

test-mobile: ## Run mobile module tests
	pnpm --dir mobile run test

check-contracts: ## Check shared API contracts
	node scripts/check-contracts.mjs

build: build-backend build-web ## Build backend binary and frontend assets

build-backend: ## Build the subcult-os backend binary
	mkdir -p bin
	cd backend && go build -o ../$(BACKEND_BIN) ./cmd/app

build-web: ## Build the frontend assets
	pnpm --dir web run build

run-backend: build-backend ## Run the local backend binary
	./$(BACKEND_BIN)

dev-mobile: ## Start the Expo mobile app
	EXPO_PUBLIC_API_URL=http://10.0.0.50:38080 pnpm --dir mobile run start --clear

dev: ## Start the full subcult-os stack
	docker compose -p $(COMPOSE_PROJECT_NAME) up

up: ## Start the full subcult-os stack in the background
	docker compose -p $(COMPOSE_PROJECT_NAME) up -d

up-build: ## Rebuild and start the full subcult-os stack in the background
	docker compose -p $(COMPOSE_PROJECT_NAME) up -d --build

down: ## Stop the subcult-os stack
	docker compose -p $(COMPOSE_PROJECT_NAME) down

reset-db: ## Stop the stack, delete local Postgres data, rebuild, and restart
	docker compose -p $(COMPOSE_PROJECT_NAME) down -v
	docker compose -p $(COMPOSE_PROJECT_NAME) up -d --build

restart: down up ## Restart the subcult-os stack

logs: ## Follow subcult-os stack logs
	docker compose -p $(COMPOSE_PROJECT_NAME) logs -f

ps: ## Show subcult-os stack containers
	docker compose -p $(COMPOSE_PROJECT_NAME) ps

urls: ## Print local stack URLs and published ports
	@printf "Web:      http://localhost:%s\n" "$$(docker compose -p $(COMPOSE_PROJECT_NAME) port web 80 | awk -F: '{print $$NF}')"
	@printf "API:      http://localhost:%s/api/health\n" "$$(docker compose -p $(COMPOSE_PROJECT_NAME) port api 8080 | awk -F: '{print $$NF}')"
	@printf "Postgres: localhost:%s\n" "$$(docker compose -p $(COMPOSE_PROJECT_NAME) port postgres 5432 | awk -F: '{print $$NF}')"

smoke: ## Check running Docker services and API/web health
	docker compose -p $(COMPOSE_PROJECT_NAME) ps
	API_PORT=$$(docker compose -p $(COMPOSE_PROJECT_NAME) port api 8080 | awk -F: '{print $$NF}'); \
	WEB_PORT=$$(docker compose -p $(COMPOSE_PROJECT_NAME) port web 80 | awk -F: '{print $$NF}'); \
	if [ -z "$$API_PORT" ] || [ -z "$$WEB_PORT" ]; then \
		echo "Could not resolve API or web host ports"; \
		exit 1; \
	fi; \
	api_ready=0; \
	for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30; do \
		curl -fsS http://127.0.0.1:$$API_PORT/api/health >/dev/null && api_ready=1 && break; \
		sleep 1; \
	done; \
	if [ "$$api_ready" != "1" ]; then \
		echo "API health check failed after retries"; \
		exit 1; \
	fi; \
	web_ready=0; \
	for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30; do \
		curl -fsS http://127.0.0.1:$$WEB_PORT/ >/dev/null && web_ready=1 && break; \
		sleep 1; \
	done; \
	if [ "$$web_ready" != "1" ]; then \
		echo "Web health check failed after retries"; \
		exit 1; \
	fi

alpha-qa: ## Run end-to-end alpha lifecycle QA against the running stack
	bash scripts/alpha-qa.sh

alpha-qa-paid: ## Run paid alpha lifecycle QA against the running stack
	bash scripts/alpha-qa.sh --paid

fake-event-qa: ## Run organizer fake-event rehearsal QA against the running stack
	bash scripts/fake-event-qa.sh

compose-config: ## Validate Docker Compose config
	docker compose -p $(COMPOSE_PROJECT_NAME) config --quiet

db-shell: ## Open a psql shell in the Postgres container
	docker compose -p $(COMPOSE_PROJECT_NAME) exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

migrate: ## Apply ordered migrations to local Postgres, starting it if needed
	docker compose -p $(COMPOSE_PROJECT_NAME) up -d --wait postgres
	docker compose -p $(COMPOSE_PROJECT_NAME) build api
	docker compose -p $(COMPOSE_PROJECT_NAME) run --rm --no-deps api /app/migrate

migrate-status: ## Show local Postgres migration ledger, tables, and indexes
	docker compose -p $(COMPOSE_PROJECT_NAME) exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "select version, name, checksum, applied_at from schema_migrations order by version" -c "\dt public.*" -c "\di public.*"'

migrate-reset: reset-db migrate ## Reset local Postgres data, restart stack, and apply ordered migrations

clean: ## Remove local build outputs
	rm -rf bin web/dist mobile/.expo mobile/dist

open-pilot-check: ## Verify Open Pilot issue template exists
	@test -f .gitea/ISSUE_TEMPLATE/open-pilot-task.yaml
	@grep -q 'Test Command' .gitea/ISSUE_TEMPLATE/open-pilot-task.yaml
