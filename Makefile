# Billing Service Makefile
# Development tools and commands

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GORUN=$(GOCMD) run
GOLINT=golangci-lint

# Binary names
BINARY_NAME=billing-service
BINARY_UNIX=$(BINARY_NAME)_unix

# Main files
MAIN_FILE=cmd/server/main.go
GO_FILES=$(shell find . -name '*.go' | grep -v test | grep -v vendor)

# Air config for auto-reload
AIR_CONFIG=./air.toml

.PHONY: all build clean test install-deps dev run test-with-coverage fmt lint help

# Default target
all: install-deps fmt lint test build

## Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_FILE)

## Clean build files
clean:
	@echo "Cleaning build files..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

## Install development dependencies
install-deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@if ! command -v air >/dev/null 2>&1; then \
		echo "Installing air for auto-reload..."; \
		$(GOGET) -u github.com/cosmtrek/air; \
	fi

## Run with auto-reload (development mode)
dev: install-deps
	@echo "Starting development server with auto-reload..."
	@if [ -f $(AIR_CONFIG) ]; then \
		air; \
	else \
		$(GORUN) $(MAIN_FILE); \
	fi

## Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	$(GORUN) $(MAIN_FILE)

## Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## Run tests with coverage
test-with-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	@echo "Coverage report:"
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Format Go code
fmt:
	@echo "Formatting Go code..."
	$(GOFMT) ./...

## Run linter
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$GOPATH/bin v1.54.2"; \
	fi

## Install specific versions of dependencies
install-stripe:
	@echo "Installing Stripe dependencies..."
	$(GOGET) github.com/stripe/stripe-go/v72@latest

## Database operations
db-migrate:
	@echo "Database migrations would go here..."
	# Add your migration commands here

db-reset:
	@echo "Resetting database..."
	# Add your database reset commands here

## Docker operations
docker-build:
	@echo "Building Docker image..."
	docker build -t billing-service .

docker-run:
	@echo "Running with Docker Compose..."
	docker-compose up --build

docker-stop:
	@echo "Stopping Docker containers..."
	docker-compose down

## Health check
health:
	@echo "Checking service health..."
	@curl -f http://localhost:9000/health || echo "Service not responding on port 9000"

## Development workflow
watch:
	@echo "Starting file watcher for auto-reload..."
	@if command -v fswatch >/dev/null 2>&1; then \
		fswatch -r . --exclude='.*' --exclude='vendor' --ext go '$(GORUN) $(MAIN_FILE)'; \
	else \
		echo "fswatch not found. Install with: brew install fswatch (macOS) or apt install fswatch (Linux)"; \
	fi

## Show help
help:
	@echo "Available commands:"
	@echo "  make install-deps   - Install development dependencies"
	@echo "  make dev            - Run with auto-reload (recommended)"
	@echo "  make run            - Run the application"
	@echo "  make build          - Build the binary"
	@echo "  make test           - Run tests"
	@echo "  make test-with-coverage - Run tests with coverage report"
	@echo "  make fmt            - Format Go code"
	@echo "  make lint           - Run linter"
	@echo "  make clean          - Clean build files"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run with Docker Compose"
	@echo "  make health         - Check service health"
	@echo "  make help           - Show this help"

## Quick start for new developers
setup: install-deps
	@echo "Setting up development environment..."
	@echo "Environment configured for port 9000"
	@echo "Run 'make dev' to start development server"