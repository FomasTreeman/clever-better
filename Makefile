# =============================================================================
# Clever Better - Makefile
# =============================================================================

# Configuration
GO_VERSION := 1.22
PYTHON_VERSION := 3.11
DOCKER_REGISTRY := your-registry
PROJECT_NAME := clever-better

# Database configuration
DB_HOST := localhost
DB_PORT := 5432
DB_NAME := clever_better
DB_USER := postgres
DB_PASSWORD := postgres
DB_URL := postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

# =============================================================================
# Database Targets
# =============================================================================

.PHONY: db-create
db-create: ## Create the database
	@echo "Creating database '$(DB_NAME)'..."
	@psql -h $(DB_HOST) -U $(DB_USER) -tc "SELECT 1 FROM pg_database WHERE datname = '$(DB_NAME)'" | grep -q 1 || psql -h $(DB_HOST) -U $(DB_USER) -c "CREATE DATABASE $(DB_NAME);"
	@echo "Database '$(DB_NAME)' created or already exists"

.PHONY: db-drop
db-drop: ## Drop the database
	@echo "Dropping database '$(DB_NAME)'..."
	@psql -h $(DB_HOST) -U $(DB_USER) -c "DROP DATABASE IF EXISTS $(DB_NAME);"
	@echo "Database '$(DB_NAME)' dropped"

.PHONY: db-reset
db-reset: db-drop db-create ## Drop and recreate the database

.PHONY: db-migrate-up
db-migrate-up: ## Run all migrations
	@echo "Running migrations..."
	@migrate -path migrations -database "$(DB_URL)" up
	@echo "Migrations completed"

.PHONY: db-migrate-down
db-migrate-down: ## Rollback all migrations
	@echo "Rolling back migrations..."
	@migrate -path migrations -database "$(DB_URL)" down
	@echo "Migrations rolled back"

.PHONY: db-migrate-status
db-migrate-status: ## Show migration status
	@migrate -path migrations -database "$(DB_URL)" version

.PHONY: db-migrate-create
db-migrate-create: ## Create a new migration (use: make db-migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME not specified. Use: make db-migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@echo "Creating migration '$(NAME)'..."
	@migrate create -ext sql -dir migrations -seq $(NAME)

.PHONY: db-migrate-force
db-migrate-force: ## Force migration version (use: make db-migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION not specified. Use: make db-migrate-force VERSION=1"; \
		exit 1; \
	fi
	@echo "Forcing migration to version $(VERSION)..."
	@migrate -path migrations -database "$(DB_URL)" force $(VERSION)

.PHONY: db-health-check
db-health-check: ## Check database connection
	@echo "Checking database connection..."
	@psql -h $(DB_HOST) -U $(DB_USER) -d $(DB_NAME) -c "SELECT 1;" && echo "✓ Database connection OK" || echo "✗ Database connection FAILED"

.PHONY: db-setup
db-setup: db-create db-migrate-up ## Create database and run migrations

# =============================================================================
# Go Targets
# =============================================================================

.PHONY: go-deps
go-deps: ## Download Go dependencies
	go mod download
	go mod verify

.PHONY: go-tidy
go-tidy: ## Tidy Go modules
	go mod tidy

.PHONY: go-build
go-build: ## Build all Go binaries
	@mkdir -p bin
	go build -o bin/bot ./cmd/bot
	go build -o bin/backtest ./cmd/backtest
	go build -o bin/data-ingestion ./cmd/data-ingestion

.PHONY: go-test
go-test: ## Run Go tests
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: go-test-short
go-test-short: ## Run Go tests (short mode)
	go test -v -short ./...

.PHONY: go-coverage
go-coverage: go-test ## Generate Go test coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: go-lint
go-lint: ## Run Go linters
	golangci-lint run ./...

.PHONY: go-fmt
go-fmt: ## Format Go code
	go fmt ./...
	goimports -w .

.PHONY: go-vet
go-vet: ## Run Go vet
	go vet ./...

.PHONY: go-sec
go-sec: ## Run Go security scanner
	gosec ./...

# =============================================================================
# Python Targets
# =============================================================================

.PHONY: py-venv
py-venv: ## Create Python virtual environment
	cd ml-service && python$(PYTHON_VERSION) -m venv venv

.PHONY: py-deps
py-deps: ## Install Python dependencies
	cd ml-service && pip install -r requirements.txt

.PHONY: py-deps-dev
py-deps-dev: ## Install Python development dependencies
	cd ml-service && pip install -r requirements-dev.txt

.PHONY: py-test
py-test: ## Run Python tests
	cd ml-service && pytest tests/ -v

.PHONY: py-test-cov
py-test-cov: ## Run Python tests with coverage
	cd ml-service && pytest tests/ -v --cov=app --cov-report=html

.PHONY: py-lint
py-lint: ## Run Python linters
	cd ml-service && flake8 app/
	cd ml-service && mypy app/

.PHONY: proto-gen
proto-gen: proto-gen-go proto-gen-python ## Generate gRPC code for ML service (Go + Python)

.PHONY: proto-gen-python
proto-gen-python: ## Generate Python gRPC code only
	cd ml-service && python -m grpc_tools.protoc -I proto --python_out=app/generated --grpc_python_out=app/generated proto/ml_service.proto

.PHONY: proto-gen-go
proto-gen-go: ## Generate Go gRPC code only
	cd ml-service/proto && protoc --go_out=../../internal/ml/mlpb --go-grpc_out=../../internal/ml/mlpb --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ml_service.proto

.PHONY: ml-test
ml-test: ## Run ML service tests
	$(MAKE) py-test

.PHONY: ml-lint
ml-lint: ## Run ML service linters
	$(MAKE) py-lint

.PHONY: py-fmt
py-fmt: ## Format Python code
	cd ml-service && black app/ tests/
	cd ml-service && isort app/ tests/

.PHONY: py-sec
py-sec: ## Run Python security scanner
	cd ml-service && safety check -r requirements.txt
	cd ml-service && bandit -r app/

# =============================================================================
# Docker Targets
# =============================================================================

.PHONY: docker-build
docker-build: ## Build Docker images
	docker build -t $(PROJECT_NAME)-bot:latest -f Dockerfile .
	docker build -t $(PROJECT_NAME)-ml:latest -f ml-service/Dockerfile ml-service/

.PHONY: docker-build-bot
docker-build-bot: ## Build bot Docker image
	docker build -t $(PROJECT_NAME)-bot:latest -f Dockerfile .

.PHONY: docker-build-ml
docker-build-ml: ## Build ML service Docker image
	docker build -t $(PROJECT_NAME)-ml:latest -f ml-service/Dockerfile ml-service/

.PHONY: docker-up
docker-up: ## Start local development environment
	docker-compose up -d

.PHONY: docker-down
docker-down: ## Stop local development environment
	docker-compose down

.PHONY: docker-logs
docker-logs: ## View Docker logs
	docker-compose logs -f

.PHONY: docker-ps
docker-ps: ## Show running containers
	docker-compose ps

.PHONY: docker-clean
docker-clean: ## Remove Docker images and volumes
	docker-compose down -v --rmi local

# =============================================================================
# Terraform Targets
# =============================================================================

.PHONY: tf-init
tf-init: ## Initialize Terraform
	cd terraform/environments/dev && terraform init

.PHONY: tf-plan
tf-plan: ## Run Terraform plan
	cd terraform/environments/dev && terraform plan

.PHONY: tf-apply
tf-apply: ## Apply Terraform changes
	cd terraform/environments/dev && terraform apply

.PHONY: tf-destroy
tf-destroy: ## Destroy Terraform infrastructure
	cd terraform/environments/dev && terraform destroy

.PHONY: tf-fmt
tf-fmt: ## Format Terraform files
	terraform fmt -recursive terraform/

.PHONY: tf-validate
tf-validate: ## Validate Terraform configuration
	cd terraform/environments/dev && terraform validate

# =============================================================================
# Database Targets
# =============================================================================

.PHONY: db-migrate-up
db-migrate-up: ## Run database migrations
	migrate -path migrations -database "$(DB_URL)" up

.PHONY: db-migrate-down
db-migrate-down: ## Rollback last database migration
	migrate -path migrations -database "$(DB_URL)" down 1

.PHONY: db-migrate-create
db-migrate-create: ## Create a new migration (usage: make db-migrate-create NAME=migration_name)
	migrate create -ext sql -dir migrations -seq $(NAME)

.PHONY: db-reset
db-reset: ## Reset database (drop and recreate)
	@echo "WARNING: This will delete all data!"
	migrate -path migrations -database "$(DB_URL)" drop -f
	migrate -path migrations -database "$(DB_URL)" up

# =============================================================================
# Testing Targets
# =============================================================================

.PHONY: test
test: go-test py-test ## Run all tests

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -v -tags=integration ./test/integration/...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	go test -v -tags=e2e ./test/e2e/...

.PHONY: test-all
test-all: test test-integration test-e2e ## Run all test suites

# =============================================================================
# Linting and Formatting
# =============================================================================

.PHONY: lint
lint: go-lint py-lint tf-validate ## Run all linters

.PHONY: fmt
fmt: go-fmt py-fmt tf-fmt ## Format all code

.PHONY: check
check: lint test ## Run linters and tests

# =============================================================================
# Build and Run
# =============================================================================

.PHONY: build
build: go-build ## Build all components

.PHONY: run-bot
run-bot: ## Run the trading bot locally
	go run ./cmd/bot

.PHONY: run-backtest
run-backtest: ## Run backtesting tool
	go run ./cmd/backtest

.PHONY: run-data-ingestion
run-data-ingestion: ## Run data ingestion service
	go run ./cmd/data-ingestion

.PHONY: run-ml
run-ml: ## Run ML service locally
	cd ml-service && (python -m app.grpc_server & uvicorn app.main:app --reload --host 0.0.0.0 --port 8000)

# =============================================================================
# Development Tools
# =============================================================================

.PHONY: install-tools
install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	pip install black isort flake8 mypy pytest safety bandit

.PHONY: setup
setup: install-tools go-deps py-deps ## Set up development environment

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out
	rm -f coverage.html
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete 2>/dev/null || true
	find . -type d -name ".pytest_cache" -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name ".mypy_cache" -exec rm -rf {} + 2>/dev/null || true

.PHONY: clean-all
clean-all: clean docker-clean ## Clean everything including Docker

# =============================================================================
# ML Integration Targets
# =============================================================================

.PHONY: ml-feedback
ml-feedback: ## Run ML feedback submission
	go run cmd/ml-feedback/main.go

.PHONY: strategy-discovery
strategy-discovery: ## Run strategy discovery pipeline
	go run cmd/strategy-discovery/main.go

.PHONY: ml-status
ml-status: ## Check ML service health and status
	go run cmd/ml-status/main.go

.PHONY: test-ml-integration
test-ml-integration: ## Run ML integration tests
	go test -v ./test/integration/ml_integration_test.go

# =============================================================================
# Documentation
# =============================================================================

.PHONY: docs-serve
docs-serve: ## Serve documentation locally
	@echo "Serving documentation at http://localhost:8000"
	@cd docs && python -m http.server 8000

.PHONY: docs-diagrams
docs-diagrams: ## Generate diagrams from Mermaid files
	@echo "Generating diagrams from Mermaid sources..."
	@for file in docs/diagrams/*.mmd; do \
		mmdc -i $$file -o $${file%.mmd}.png; \
	done

# =============================================================================
# Security
# =============================================================================

.PHONY: security
security: go-sec py-sec ## Run all security scanners

.PHONY: audit
audit: ## Audit dependencies for vulnerabilities
	go list -json -m all | nancy sleuth
	cd ml-service && safety check -r requirements.txt
