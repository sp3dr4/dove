.PHONY: help dev run docker-up docker-down build build-all docker-build docker-run clean test test-verbose test-coverage test-integration test-all test-race lint fmt vet mod-tidy mod-verify swagger migrate-create check install-tools

DEFAULT_GOAL := help

APP_NAME := dove
GO_VERSION := 1.24
DOCKER_IMAGE := $(APP_NAME):latest
BUILD_DIR := ./bin
MAIN_PATH := ./cmd/server/main.go
SERVER_PATH := ./cmd/server/main.go
CLI_PATH := ./cmd/cli/main.go

COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m


help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

dev: ## Start development server with hot reload
	@echo "$(COLOR_BLUE)Starting development server with hot reload...$(COLOR_RESET)"
	@air -c .air.toml

run: ## Run the application directly
	@echo "$(COLOR_BLUE)Running application...$(COLOR_RESET)"
	@go run $(MAIN_PATH)

docker-up: ## Start development dependencies
	@echo "$(COLOR_BLUE)Starting development dependencies...$(COLOR_RESET)"
	@docker compose up -d
	@echo "$(COLOR_GREEN)Development environment is ready!$(COLOR_RESET)"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"

docker-down: ## Stop development dependencies
	@echo "$(COLOR_BLUE)Stopping development dependencies...$(COLOR_RESET)"
	@docker compose down

build: ## Build the application binary
	@echo "$(COLOR_BLUE)Building application...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(COLOR_GREEN)Build complete: $(BUILD_DIR)/$(APP_NAME)$(COLOR_RESET)"

build-all: ## Build server and CLI binaries
	@echo "$(COLOR_BLUE)Building all binaries...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/server $(SERVER_PATH) 2>/dev/null || echo "$(COLOR_YELLOW)Server binary not yet implemented$(COLOR_RESET)"
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/cli $(CLI_PATH) 2>/dev/null || echo "$(COLOR_YELLOW)CLI binary not yet implemented$(COLOR_RESET)"
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(COLOR_GREEN)All builds complete!$(COLOR_RESET)"

docker-build: ## Build Docker image
	@echo "$(COLOR_BLUE)Building Docker image...$(COLOR_RESET)"
	@docker build -t $(DOCKER_IMAGE) .
	@echo "$(COLOR_GREEN)Docker image built: $(DOCKER_IMAGE)$(COLOR_RESET)"

docker-run: docker-build ## Run the Docker image
	@echo "$(COLOR_BLUE)Running Docker image...$(COLOR_RESET)"
	@docker run --rm -p 8080:8080 --name $(APP_NAME)-container $(DOCKER_IMAGE)

clean: ## Clean build artifacts
	@echo "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -rf tmp/
	@rm -rf coverage/
	@rm -f coverage.out
	@rm -f coverage.html
	@echo "$(COLOR_GREEN)Clean complete!$(COLOR_RESET)"

test: ## Run unit tests (fast feedback)
	@echo "$(COLOR_BLUE)Running unit tests...$(COLOR_RESET)"
	@go test -short ./...

test-all: ## Run all tests with coverage report
	@echo "$(COLOR_BLUE)Running all tests with coverage...$(COLOR_RESET)"
	@mkdir -p coverage
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./... ./test/integration/...
	@go tool cover -html=coverage.out -o coverage/index.html
	@echo "$(COLOR_GREEN)Coverage report generated: coverage/index.html$(COLOR_RESET)"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total Coverage: " $$3}'

lint: fmt vet ## Run golangci-lint (with formatting and vet)
	@echo "$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@golangci-lint run ./...

fmt: ## Format code
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	@go fmt ./...
	@goimports -w . 2>/dev/null || echo "$(COLOR_YELLOW)Run 'make install-tools' to install goimports$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)Formatting complete!$(COLOR_RESET)"

vet: ## Run go vet
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	@go vet ./...

mod-tidy: ## Tidy go modules
	@echo "$(COLOR_BLUE)Tidying go modules...$(COLOR_RESET)"
	@go mod tidy

mod-verify: mod-tidy ## Verify go modules (with tidy)
	@echo "$(COLOR_BLUE)Verifying go modules...$(COLOR_RESET)"
	@go mod verify

swagger: ## Generate Swagger documentation
	@echo "$(COLOR_BLUE)Generating Swagger documentation...$(COLOR_RESET)"
	@swag init -g cmd/server/main.go --output docs --parseDependency --parseInternal
	@swag fmt
	@echo "$(COLOR_GREEN)Swagger documentation generated!$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)Access Swagger UI at: http://localhost:8080/swagger/index.html$(COLOR_RESET)"

migrate-create: ## Create migration for both databases
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations/postgres -seq $$name || echo "$(COLOR_YELLOW)Install migrate tool: https://github.com/golang-migrate/migrate$(COLOR_RESET)"; \
	migrate create -ext sql -dir migrations/sqlite -seq $$name || echo "$(COLOR_YELLOW)Install migrate tool: https://github.com/golang-migrate/migrate$(COLOR_RESET)"

check: lint test ## Run all quality checks (format, vet, lint, test)

install-tools: ## Install development tools
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	@go install github.com/air-verse/air@latest
	@echo "$(COLOR_BLUE)Installing golangci-lint via official installer...$(COLOR_RESET)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/vektra/mockery/v2@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "$(COLOR_YELLOW)Note: Install migrate manually from https://github.com/golang-migrate/migrate$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)Tools installed successfully!$(COLOR_RESET)"

# Create build directory if it doesn't exist
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)
