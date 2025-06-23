# Enterprise E-commerce Backend Makefile

# Variables
DOCKER_COMPOSE = docker-compose
GO_CMD = go
BINARY_NAME = main
MAIN_PATH = cmd/api/main.go
PKG_LIST = ./...

# Colors for output
BLUE = \033[36m
GREEN = \033[32m
YELLOW = \033[33m
RED = \033[31m
NC = \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo '$(BLUE)Enterprise E-commerce Backend$(NC)'
	@echo '$(BLUE)==============================$(NC)'
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(GREEN)<target>$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Setup
.PHONY: setup
setup: ## Initial project setup
	@echo "$(BLUE)Setting up project...$(NC)"
	$(GO_CMD) mod tidy
	cp .env.example .env
	@echo "$(GREEN)✅ Setup completed!$(NC)"
	@echo "$(YELLOW)📝 Please edit .env file with your configuration$(NC)"

##@ Development
.PHONY: dev
dev: ## Start development environment
	@echo "$(BLUE)Starting development environment...$(NC)"
	$(DOCKER_COMPOSE) up --build

.PHONY: dev-down
dev-down: ## Stop development environment
	@echo "$(YELLOW)Stopping development environment...$(NC)"
	$(DOCKER_COMPOSE) down

.PHONY: logs
logs: ## Show application logs
	$(DOCKER_COMPOSE) logs -f backend

.PHONY: shell
shell: ## Access backend container shell
	$(DOCKER_COMPOSE) exec backend sh

##@ Tools
.PHONY: tools
tools: ## Start additional tools (adminer, redis-commander)
	@echo "$(BLUE)Starting development tools...$(NC)"
	$(DOCKER_COMPOSE) --profile tools up -d adminer redis-commander

.PHONY: tools-down
tools-down: ## Stop additional tools
	$(DOCKER_COMPOSE) --profile tools down

##@ Build
.PHONY: build
build: ## Build the application
	@echo "$(BLUE)Building application...$(NC)"
	$(GO_CMD) build -o $(BINARY_NAME) $(MAIN_PATH)

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	$(GO_CMD) clean
	rm -f $(BINARY_NAME)
	rm -rf tmp/

.PHONY: db-shell
db-shell: ## Access database shell
	$(DOCKER_COMPOSE) exec postgres psql -U ecommerce_user -d ecommerce_db
