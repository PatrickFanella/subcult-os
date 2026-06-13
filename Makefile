SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

PROJECT_NAME := subcult-os
COMPOSE_PROJECT_NAME ?= $(PROJECT_NAME)
BACKEND_BIN ?= bin/$(PROJECT_NAME)

.PHONY: help deps verify quick fmt lint test test-backend test-web build build-backend build-web run-backend dev up up-build down restart logs ps compose-config db-shell clean open-pilot-check

help:
	@awk 'BEGIN {FS = ":.*##"; printf "$(PROJECT_NAME) commands:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install frontend dependencies
	pnpm --dir web install --frozen-lockfile

verify: deps fmt lint test build compose-config open-pilot-check ## Run all checks

quick: fmt lint test ## Run fast local checks without Docker config or production builds

fmt: ## Format/check Go and TypeScript sources
	@git ls-files --cached --others --exclude-standard 'backend/**/*.go' 'backend/*.go' | xargs -r gofmt -w
	pnpm --dir web run format

lint: ## Run Go vet and frontend lint
	cd backend && go vet ./...
	pnpm --dir web run lint

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

dev: ## Start the full subcult-os stack
	docker compose -p $(COMPOSE_PROJECT_NAME) up

up: ## Start the full subcult-os stack in the background
	docker compose -p $(COMPOSE_PROJECT_NAME) up -d

up-build: ## Rebuild and start the full subcult-os stack in the background
	docker compose -p $(COMPOSE_PROJECT_NAME) up -d --build

down: ## Stop the subcult-os stack
	docker compose -p $(COMPOSE_PROJECT_NAME) down

restart: down up ## Restart the subcult-os stack

logs: ## Follow subcult-os stack logs
	docker compose -p $(COMPOSE_PROJECT_NAME) logs -f

ps: ## Show subcult-os stack containers
	docker compose -p $(COMPOSE_PROJECT_NAME) ps

compose-config: ## Validate Docker Compose config
	docker compose -p $(COMPOSE_PROJECT_NAME) config --quiet

db-shell: ## Open a psql shell in the Postgres container
	docker compose -p $(COMPOSE_PROJECT_NAME) exec postgres psql -U $${POSTGRES_USER:-app} -d $${POSTGRES_DB:-app}

clean: ## Remove local build outputs
	rm -rf bin web/dist

open-pilot-check: ## Verify Open Pilot issue template exists
	@test -f .gitea/ISSUE_TEMPLATE/open-pilot-task.yaml
	@grep -q 'Test Command' .gitea/ISSUE_TEMPLATE/open-pilot-task.yaml
