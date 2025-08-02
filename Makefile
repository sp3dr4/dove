.PHONY: help dev test lint build clean docker-dev docker-build install-tools swagger swagger-fmt

# Default target
DEFAULT_GOAL := help

# Variables
APP_NAME := dove
GO_VERSION := 1.24
DOCKER_IMAGE := $(APP_NAME):latest
BUILD_DIR := ./bin
MAIN_PATH := ./cmd/server/main.go
SERVER_PATH := ./cmd/server/main.go
CLI_PATH := ./cmd/cli/main.go

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

## help: Show this help message
help:
	@echo "$(COLOR_BOLD)$(APP_NAME) Development Commands$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Usage:$(COLOR_RESET)"
	@echo "  make $(COLOR_GREEN)<target>$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Development:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)dev$(COLOR_RESET)             Start development server with hot reload"
	@echo "  $(COLOR_GREEN)run$(COLOR_RESET)             Run the application directly"
	@echo "  $(COLOR_GREEN)docker-dev$(COLOR_RESET)      Start development dependencies (PostgreSQL, Redis)"
	@echo "  $(COLOR_GREEN)docker-down$(COLOR_RESET)     Stop development dependencies"
	@echo ""
	@echo "$(COLOR_BOLD)Building:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)build$(COLOR_RESET)           Build the application binary"
	@echo "  $(COLOR_GREEN)build-all$(COLOR_RESET)       Build server and CLI binaries"
	@echo "  $(COLOR_GREEN)docker-build$(COLOR_RESET)    Build Docker image"
	@echo "  $(COLOR_GREEN)clean$(COLOR_RESET)           Clean build artifacts"
	@echo ""
	@echo "$(COLOR_BOLD)Testing:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)test$(COLOR_RESET)            Run unit tests"
	@echo "  $(COLOR_GREEN)test-verbose$(COLOR_RESET)    Run tests with verbose output"
	@echo "  $(COLOR_GREEN)test-coverage$(COLOR_RESET)   Run tests with coverage report"
	@echo "  $(COLOR_GREEN)test-integration$(COLOR_RESET) Run integration tests"
	@echo "  $(COLOR_GREEN)test-all$(COLOR_RESET)        Run all tests with coverage"
	@echo "  $(COLOR_GREEN)test-race$(COLOR_RESET)       Run tests with race detector"
	@echo ""
	@echo "$(COLOR_BOLD)Code Quality:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)lint$(COLOR_RESET)            Run golangci-lint"
	@echo "  $(COLOR_GREEN)fmt$(COLOR_RESET)             Format code with gofmt and goimports"
	@echo "  $(COLOR_GREEN)vet$(COLOR_RESET)             Run go vet"
	@echo "  $(COLOR_GREEN)mod-tidy$(COLOR_RESET)        Tidy go modules"
	@echo "  $(COLOR_GREEN)mod-verify$(COLOR_RESET)      Verify go modules"
	@echo ""
	@echo "$(COLOR_BOLD)Documentation:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)swagger$(COLOR_RESET)         Generate Swagger documentation"
	@echo "  $(COLOR_GREEN)swagger-fmt$(COLOR_RESET)     Format Swagger annotations"
	@echo ""
	@echo "$(COLOR_BOLD)Database:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)migrate-up$(COLOR_RESET)      Run database migrations up"
	@echo "  $(COLOR_GREEN)migrate-down$(COLOR_RESET)    Run database migrations down"
	@echo "  $(COLOR_GREEN)migrate-create$(COLOR_RESET)  Create a new migration"
	@echo ""
	@echo "$(COLOR_BOLD)Tools:$(COLOR_RESET)"
	@echo "  $(COLOR_GREEN)install-tools$(COLOR_RESET)   Install development tools"
	@echo "  $(COLOR_GREEN)update-tools$(COLOR_RESET)    Update development tools"
	@echo ""

## dev: Start development server with hot reload
dev:
	@echo "$(COLOR_BLUE)Starting development server with hot reload...$(COLOR_RESET)"
	@air -c .air.toml

## run: Run the application directly
run:
	@echo "$(COLOR_BLUE)Running application...$(COLOR_RESET)"
	@go run $(MAIN_PATH)

## docker-dev: Start development dependencies
docker-dev:
	@echo "$(COLOR_BLUE)Starting development dependencies...$(COLOR_RESET)"
	@docker compose up -d
	@echo "$(COLOR_GREEN)Development environment is ready!$(COLOR_RESET)"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"

## docker-down: Stop development dependencies
docker-down:
	@echo "$(COLOR_BLUE)Stopping development dependencies...$(COLOR_RESET)"
	@docker compose down

## build: Build the application binary
build:
	@echo "$(COLOR_BLUE)Building application...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(COLOR_GREEN)Build complete: $(BUILD_DIR)/$(APP_NAME)$(COLOR_RESET)"

## build-all: Build server and CLI binaries
build-all:
	@echo "$(COLOR_BLUE)Building all binaries...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/server $(SERVER_PATH) 2>/dev/null || echo "$(COLOR_YELLOW)Server binary not yet implemented$(COLOR_RESET)"
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/cli $(CLI_PATH) 2>/dev/null || echo "$(COLOR_YELLOW)CLI binary not yet implemented$(COLOR_RESET)"
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(COLOR_GREEN)All builds complete!$(COLOR_RESET)"

## docker-build: Build Docker image
docker-build:
	@echo "$(COLOR_BLUE)Building Docker image...$(COLOR_RESET)"
	@docker build -t $(DOCKER_IMAGE) .
	@echo "$(COLOR_GREEN)Docker image built: $(DOCKER_IMAGE)$(COLOR_RESET)"

## clean: Clean build artifacts
clean:
	@echo "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -rf tmp/
	@rm -rf coverage/
	@rm -f coverage.out
	@rm -f coverage.html
	@echo "$(COLOR_GREEN)Clean complete!$(COLOR_RESET)"

## test: Run unit tests
test:
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	@go test -short ./...

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "$(COLOR_BLUE)Running tests (verbose)...$(COLOR_RESET)"
	@go test -v -short ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "$(COLOR_BLUE)Running tests with coverage...$(COLOR_RESET)"
	@mkdir -p coverage
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage/index.html
	@echo "$(COLOR_GREEN)Coverage report generated: coverage/index.html$(COLOR_RESET)"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total Coverage: " $$3}'

## test-integration: Run integration tests
test-integration:
	@echo "$(COLOR_BLUE)Running integration tests...$(COLOR_RESET)"
	@go test -v -tags=integration ./...

## test-all: Run all tests with coverage
test-all:
	@echo "$(COLOR_BLUE)Running all tests...$(COLOR_RESET)"
	@go test -v -race -tags=integration -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage/index.html
	@echo "$(COLOR_GREEN)Coverage report generated: coverage/index.html$(COLOR_RESET)"

## test-race: Run tests with race detector
test-race:
	@echo "$(COLOR_BLUE)Running tests with race detector...$(COLOR_RESET)"
	@go test -race ./...

## lint: Run golangci-lint
lint:
	@echo "$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	@go fmt ./...
	@goimports -w . 2>/dev/null || echo "$(COLOR_YELLOW)Run 'make install-tools' to install goimports$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)Formatting complete!$(COLOR_RESET)"

## vet: Run go vet
vet:
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	@go vet ./...

## mod-tidy: Tidy go modules
mod-tidy:
	@echo "$(COLOR_BLUE)Tidying go modules...$(COLOR_RESET)"
	@go mod tidy

## mod-verify: Verify go modules
mod-verify:
	@echo "$(COLOR_BLUE)Verifying go modules...$(COLOR_RESET)"
	@go mod verify

## swagger: Generate Swagger documentation
swagger:
	@echo "$(COLOR_BLUE)Generating Swagger documentation...$(COLOR_RESET)"
	@swag init -g cmd/server/main.go --output docs --parseDependency --parseInternal
	@echo "$(COLOR_GREEN)Swagger documentation generated!$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)Access Swagger UI at: http://localhost:8080/swagger/index.html$(COLOR_RESET)"

## swagger-fmt: Format Swagger annotations
swagger-fmt:
	@echo "$(COLOR_BLUE)Formatting Swagger annotations...$(COLOR_RESET)"
	@swag fmt
	@echo "$(COLOR_GREEN)Swagger annotations formatted!$(COLOR_RESET)"

## migrate-up: Run database migrations up
migrate-up:
	@echo "$(COLOR_BLUE)Running migrations up...$(COLOR_RESET)"
	@migrate -path migrations -database "$${DATABASE_URL}" up || echo "$(COLOR_YELLOW)Install migrate tool: https://github.com/golang-migrate/migrate$(COLOR_RESET)"

## migrate-down: Run database migrations down
migrate-down:
	@echo "$(COLOR_BLUE)Running migrations down...$(COLOR_RESET)"
	@migrate -path migrations -database "$${DATABASE_URL}" down || echo "$(COLOR_YELLOW)Install migrate tool: https://github.com/golang-migrate/migrate$(COLOR_RESET)"

## migrate-create: Create a new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name || echo "$(COLOR_YELLOW)Install migrate tool: https://github.com/golang-migrate/migrate$(COLOR_RESET)"

## install-tools: Install development tools
install-tools:
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	@go install github.com/air-verse/air@latest
	@echo "$(COLOR_BLUE)Installing golangci-lint via official installer...$(COLOR_RESET)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/vektra/mockery/v2@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "$(COLOR_YELLOW)Note: Install migrate manually from https://github.com/golang-migrate/migrate$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)Tools installed successfully!$(COLOR_RESET)"

## update-tools: Update development tools
update-tools: install-tools
	@echo "$(COLOR_GREEN)Tools updated successfully!$(COLOR_RESET)"

# Database URL check
check-db-url:
ifndef DATABASE_URL
	$(error DATABASE_URL is not set. Please set it in your environment or .env file)
endif

# Ensure migrations exist before running them
migrate-up migrate-down: check-db-url

# Create build directory if it doesn't exist
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)
