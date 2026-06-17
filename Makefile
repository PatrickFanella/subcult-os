SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

PROJECT_NAME := subcult-os
COMPOSE_PROJECT_NAME ?= $(PROJECT_NAME)
BACKEND_BIN ?= bin/$(PROJECT_NAME)

.PHONY: help deps deps-web deps-mobile verify quick fmt lint lint-mobile test test-backend test-web build build-backend build-web run-backend dev dev-mobile up up-build down reset-db restart logs ps urls smoke alpha-qa alpha-qa-paid compose-config db-shell migrate migrate-status migrate-reset clean open-pilot-check

help:
	@awk 'BEGIN {FS = ":.*##"; printf "$(PROJECT_NAME) commands:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: deps-web deps-mobile ## Install web and mobile dependencies

deps-web: ## Install web dependencies
	pnpm --dir web install --frozen-lockfile

deps-mobile: ## Install mobile dependencies
	pnpm --dir mobile install --frozen-lockfile

verify: deps fmt lint test build compose-config open-pilot-check ## Run all checks

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

test: test-backend test-web ## Run backend and frontend tests

test-backend: ## Run Go tests
	cd backend && go test ./...

test-web: ## Run frontend tests
	pnpm --dir web run test

build: build-backend build-web ## Build backend binary and frontend assets

build-backend: ## Build the subcult-os backend binary
	mkdir -p bin
	cd backend && go build -o ../$(BACKEND_BIN) ./cmd/app

build-web: ## Build the frontend assets
	pnpm --dir web run build

run-backend: build-backend ## Run the local backend binary
	./$(BACKEND_BIN)

dev-mobile: ## Start the Expo mobile app
	pnpm --dir mobile run start

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
	docker compose -p $(COMPOSE_PROJECT_NAME) exec -T api wget -qO- http://127.0.0.1:8080/api/health
	docker compose -p $(COMPOSE_PROJECT_NAME) exec -T web wget -qO- http://127.0.0.1/ >/dev/null

alpha-qa: ## Run end-to-end alpha lifecycle QA against the running stack
	bash scripts/alpha-qa.sh

alpha-qa-paid: ## Run paid alpha lifecycle QA against the running stack
	bash scripts/alpha-qa.sh --paid

compose-config: ## Validate Docker Compose config
	docker compose -p $(COMPOSE_PROJECT_NAME) config --quiet

db-shell: ## Open a psql shell in the Postgres container
	docker compose -p $(COMPOSE_PROJECT_NAME) exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

migrate: ## Apply the alpha schema.sql to local Postgres, starting it if needed
	docker compose -p $(COMPOSE_PROJECT_NAME) up -d --wait postgres
	docker compose -p $(COMPOSE_PROJECT_NAME) exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < backend/internal/app/schema.sql

migrate-status: ## Show local Postgres tables and applied alpha schema objects
	docker compose -p $(COMPOSE_PROJECT_NAME) exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "\dt public.*" -c "\di public.*"'

migrate-reset: reset-db migrate ## Reset local Postgres data, restart stack, and apply schema.sql

clean: ## Remove local build outputs
	rm -rf bin web/dist mobile/.expo mobile/dist

open-pilot-check: ## Verify Open Pilot issue template exists
	@test -f .gitea/ISSUE_TEMPLATE/open-pilot-task.yaml
	@grep -q 'Test Command' .gitea/ISSUE_TEMPLATE/open-pilot-task.yaml
