.PHONY: help build test test-coverage test-integration clean lint fmt docker-build docker-run docker-stop docker-clean load-test load-test-smoke load-test-quick load-test-stress dev-docker prod-sim load-test-prod security-scan security-gosec security-nancy security-all

# Default target
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  test           - Run unit tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-integration - Run integration tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker Compose services"
	@echo "  docker-clean   - Clean Docker resources"
	@echo "  load-test      - Run load tests"
	@echo "  load-test-smoke - Run smoke test"
	@echo "  load-test-quick - Run quick load test suite"
	@echo "  load-test-stress - Run stress test"
	@echo "  dev-docker     - Start development environment with Docker"
	@echo "  prod-sim       - Start production simulation"
	@echo "  load-test-prod - Run load tests against production simulation"
	@echo "  security-scan  - Run comprehensive security scan"
	@echo "  security-gosec - Run gosec security analysis"
	@echo "  security-nancy - Run nancy dependency scan"
	@echo "  security-all   - Run all security scans"

# Build the application
build:
	@echo "Building application..."
	go build -o bin/connex ./cmd/server

# Run unit tests
test:
	@echo "Running unit tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./tests/integration

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	go vet ./...

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t connex:latest .

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up --build

docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v
	docker system prune -f

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Run the application
run:
	@echo "Running application..."
	go run ./cmd/server

# Generate mocks (if using mockery)
mocks:
	@echo "Generating mocks..."
	mockery --all --output=./mocks

# Security audit
security:
	@echo "Running security audit..."
	gosec ./...
	nancy sleuth

# Performance benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	# Add your migration command here
	# Example: migrate -path internal/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	@echo "Rolling back database migrations..."
	# Add your migration command here
	# Example: migrate -path internal/db/migrations -database "$(DATABASE_URL)" down

# Development setup
dev-setup: deps migrate-up
	@echo "Development environment setup complete"

# Production build
prod-build:
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/connex ./cmd/server 

# Load testing commands
load-test:
	@echo "Running load tests..."
	chmod +x scripts/run-load-tests.sh
	./scripts/run-load-tests.sh

load-test-smoke:
	@echo "Running smoke test..."
	chmod +x scripts/run-load-tests.sh
	./scripts/run-load-tests.sh smoke

load-test-quick:
	@echo "Running quick load test suite..."
	chmod +x scripts/run-load-tests.sh
	./scripts/run-load-tests.sh quick

load-test-stress:
	@echo "Running stress test..."
	chmod +x scripts/run-load-tests.sh
	./scripts/run-load-tests.sh stress

# Development with Docker
dev-docker: docker-build
	@echo "Starting development environment with Docker..."
	docker-compose up -d postgres redis jaeger prometheus grafana
	@echo "Waiting for services to be ready..."
	sleep 10
	@echo "Starting application..."
	go run ./cmd/server

# Production simulation
prod-sim: docker-build
	@echo "Starting production simulation..."
	docker-compose -f docker-compose.yml up --build app

# Load testing with production simulation
load-test-prod: prod-sim
	@echo "Waiting for production simulation to be ready..."
	sleep 30
	@echo "Running load tests against production simulation..."
	./scripts/run-load-tests.sh 

# Security targets
security-scan:
	@echo "🔒 Running comprehensive security scan..."
	@./scripts/security-scan.sh

security-gosec:
	@echo "🔒 Running gosec security analysis..."
	@gosec ./... -fmt=json -out=gosec-report.json
	@echo "Gosec scan completed. Check gosec-report.json for details."

security-nancy:
	@echo "🔒 Running nancy dependency scan..."
	@nancy sleuth

security-all: security-gosec security-nancy
	@echo "🔒 All security scans completed."

security-test: ## Run security testing against running application
	@echo "🧪 Running security testing..."
	@./scripts/test-security.sh

security-test-unit: ## Run unit security tests
	@echo "🧪 Running unit security tests..."
	@go test -v ./tests/security/...

security-test-all: security-test-unit security-test ## Run all security tests
	@echo "✅ All security tests completed"

security-test-comprehensive: ## Run comprehensive security testing suite
	@echo "🔒 Running comprehensive security testing suite..."
	@./scripts/run-security-tests.sh 