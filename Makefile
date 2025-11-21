.PHONY: help build run import clean test

# Variables
BINARY_NAME=test-service
IMPORT_TOOL=import-tool
CONFIG_FILE=config.toml
DATA_FILE=examples/sample-tests.json

help: ## Show this help message
	@echo "Test Management Service - Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the service
	@echo "Building test management service..."
	@go build -o $(BINARY_NAME) ./cmd/server
	@echo "✓ Build complete: $(BINARY_NAME)"

build-import: ## Build the import tool
	@echo "Building import tool..."
	@go build -o $(IMPORT_TOOL) ./cmd/import
	@echo "✓ Build complete: $(IMPORT_TOOL)"

build-all: build build-import ## Build all binaries
	@echo "✓ All builds complete"

run: build ## Build and run the service
	@echo "Starting test management service..."
	@./$(BINARY_NAME)

import: build-import ## Import sample test data
	@echo "Importing test data from $(DATA_FILE)..."
	@./$(IMPORT_TOOL) -config $(CONFIG_FILE) -data $(DATA_FILE)

dev: ## Run in development mode
	@echo "Starting in development mode..."
	@GIN_MODE=debug go run ./cmd/server/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-cover: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

clean: ## Clean build artifacts and database
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME) $(IMPORT_TOOL)
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

clean-db: ## Clean database only
	@echo "Cleaning database..."
	@rm -rf data/
	@echo "✓ Database cleaned"

install-deps: ## Install Go dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies installed"

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run || echo "Note: golangci-lint not installed, skipping"

init: install-deps build-all import ## Initialize project (install deps, build, import data)
	@echo "✓ Initialization complete!"

health: ## Check service health
	@curl -s http://localhost:8090/health | json_pp || echo "Service not running"

# Quick commands for testing API
api-groups: ## List test groups
	@curl -s http://localhost:8090/api/v2/groups/tree | json_pp

api-tests: ## List test cases
	@curl -s "http://localhost:8090/api/v2/tests?limit=10" | json_pp

api-runs: ## List test runs
	@curl -s "http://localhost:8090/api/v2/runs?limit=5" | json_pp
