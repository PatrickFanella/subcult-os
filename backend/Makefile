SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

PROJECT_NAME := subcult-os
BIN ?= ../bin/$(PROJECT_NAME)

.PHONY: help verify fmt lint test build run clean

help:
	@awk 'BEGIN {FS = ":.*##"; printf "$(PROJECT_NAME) backend commands:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

verify: fmt lint test build ## Run backend checks

fmt: ## Format Go files
	gofmt -w $$(git ls-files '*.go')

lint: ## Run Go vet
	go vet ./...

test: ## Run Go tests
	go test ./...

build: ## Build the subcult-os backend binary
	mkdir -p ../bin
	go build -o $(BIN) ./cmd/app

run: build ## Run the subcult-os backend binary
	$(BIN)

clean: ## Remove backend build output
	rm -f $(BIN)
