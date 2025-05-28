# Makefile for Caddy Frontend Asset Proxy

# Default command to run when just 'make' is typed
.DEFAULT_GOAL := help

# Variables
COMPOSE_FILE := docker-compose.yml
TEST_SCRIPT := ./test_caddy.sh

# Phony targets are targets that don't represent actual files
.PHONY: help up down logs test clean clean-all setup-minio

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

up: ## Start Minio and Caddy services in detached mode
	@echo "Starting Docker Compose services (Minio and Caddy)..."
	docker-compose -f $(COMPOSE_FILE) up -d --remove-orphans
	@echo "Services started. Minio console: http://localhost:9001, Caddy proxy: http://localhost:8080"
	@echo "Run 'make setup-minio' if this is the first time or Minio data was cleared."

setup-minio: ## Remind user to configure Minio (bucket, policy, files)
	@echo "--------------------------------------------------------------------------------------"
	@echo "ACTION REQUIRED: Configure Minio (if not already done):"
	@echo "1. Go to Minio Console: http://localhost:9001 (Login: minioadmin / minioadmin)"
	@echo "2. Create bucket: 'frontend-assets'"
	@echo "3. Set 'frontend-assets' bucket Access Policy to 'Public'."
	@echo "4. Upload 'index.html' to the root of 'frontend-assets'."
	@echo "5. Upload 'edge-navigation.json' to 'frontend-assets/api/chrome-service/v1/static/stable/prod/navigation/'."
	@echo "--------------------------------------------------------------------------------------"
	@echo "Press Enter to continue after Minio setup..."
	@read

down: ## Stop Minio and Caddy services
	@echo "Stopping Docker Compose services..."
	docker-compose -f $(COMPOSE_FILE) down

logs: ## Follow logs for all services
	@echo "Following logs for all services (Ctrl+C to stop)..."
	docker-compose -f $(COMPOSE_FILE) logs -f

caddy-logs: ## Follow logs for the Caddy service
	@echo "Following logs for Caddy service (Ctrl+C to stop)..."
	docker-compose -f $(COMPOSE_FILE) logs -f caddy

minio-logs: ## Follow logs for the Minio service
	@echo "Following logs for Minio service (Ctrl+C to stop)..."
	docker-compose -f $(COMPOSE_FILE) logs -f minio

test: up setup-minio ## Start services, ensure Minio is set up, then run tests
	@echo "Running tests..."
	@if [ -x "$(TEST_SCRIPT)" ]; then \
		$(TEST_SCRIPT); \
	else \
		echo "Test script $(TEST_SCRIPT) not found or not executable. Please run 'chmod +x $(TEST_SCRIPT)'."; \
		exit 1; \
	fi
	@echo "Tests finished. Run 'make down' to stop services."

build: ## Build or rebuild the Caddy Docker image
	@echo "Building Caddy Docker image..."
	docker-compose -f $(COMPOSE_FILE) build --no-cache caddy

clean: down ## Stop and remove containers and networks
	@echo "Cleaning up (removing containers and networks)..."
	docker-compose -f $(COMPOSE_FILE) down --remove-orphans

clean-all: clean ## Stop and remove containers, networks, AND Minio data volume
	@echo "Cleaning up thoroughly (removing containers, networks, and Minio data volume)..."
	docker-compose -f $(COMPOSE_FILE) down -v --remove-orphans
	@echo "Minio data volume 'minio_data' has been removed."

