.DEFAULT_GOAL := help

GREEN := $(shell printf '\033[0;32m')
YELLOW := $(shell printf '\033[0;33m')
NC := $(shell printf '\033[0m')

.PHONY: dev
dev: build up ## Start development environment
	@echo "$(GREEN)PVE Pilot is running$(NC)"
	@echo "  Frontend: http://localhost:3000"
	@echo "  Backend:  http://localhost:8080"

.PHONY: build
build: ## Build Docker images
	docker compose build

.PHONY: up
up: ## Start containers
	docker compose up -d

.PHONY: down
down: ## Stop containers
	docker compose down

.PHONY: logs
logs: ## Show all logs
	docker compose logs -f

.PHONY: logs-api
logs-api: ## Show backend logs
	docker compose logs -f backend

.PHONY: logs-web
logs-web: ## Show frontend logs
	docker compose logs -f frontend

.PHONY: status
status: ## Show container status
	docker compose ps

.PHONY: backend-dev
backend-dev: ## Run backend locally (without Docker)
	cd backend && go run .

.PHONY: frontend-dev
frontend-dev: ## Run frontend locally (without Docker)
	cd frontend && npm run dev

.PHONY: test
test: test-backend test-frontend ## Run all tests

.PHONY: test-backend
test-backend: ## Run Go backend tests
	cd backend && go test ./... -count=1

.PHONY: test-frontend
test-frontend: ## Run frontend tests
	cd frontend && npx vitest run

.PHONY: test-verbose
test-verbose: ## Run all tests with verbose output
	cd backend && go test ./... -count=1 -v
	cd frontend && npx vitest run

.PHONY: backend-build
backend-build: ## Build backend binary
	cd backend && go build -o server .

.PHONY: clean
clean: down ## Stop and remove everything
	docker compose down -v --rmi local
	rm -f backend/server

.PHONY: help
help: ## Show this help
	@echo "$(GREEN)PVE Pilot$(NC) - Proxmox VE Dashboard"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-18s$(NC) %s\n", $$1, $$2}'
