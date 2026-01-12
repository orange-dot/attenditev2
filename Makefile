.PHONY: build run dev test clean docker-up docker-down migrate

# Build the platform binary
build:
	go build -o bin/platform ./cmd/platform

# Run the server
run: build
	./bin/platform

# Run in development mode with hot reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not installed. Install with: go install github.com/air-verse/air@latest"; \
		echo "Running without hot reload..."; \
		go run ./cmd/platform; \
	fi

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Start Docker services
docker-up:
	docker-compose up -d

# Stop Docker services
docker-down:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f

# AI Demo (minimal setup ~3GB)
demo-up:
	docker-compose -f deploy/docker/docker-compose.ai-demo.yml up -d --build

demo-down:
	docker-compose -f deploy/docker/docker-compose.ai-demo.yml down

demo-logs:
	docker-compose -f deploy/docker/docker-compose.ai-demo.yml logs -f

# Docker cleanup commands
docker-clean:
	@echo "Cleaning dangling images and build cache..."
	docker image prune -f
	docker builder prune -f

docker-clean-all:
	@echo "WARNING: Removing ALL unused images (not just dangling)..."
	docker system prune -f
	docker builder prune -af

docker-clean-nuclear:
	@echo "WARNING: Removing EVERYTHING except volumes..."
	docker system prune -af
	docker builder prune -af

# Run database migrations
migrate:
	go run ./cmd/platform migrate

# Generate OpenAPI spec
openapi:
	@echo "OpenAPI generation not yet implemented"

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Lint code
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Security scan
security:
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

# All checks before commit
check: fmt lint test

# Help
help:
	@echo "Serbia Government Interoperability Platform"
	@echo ""
	@echo "Development:"
	@echo "  make build          - Build the platform binary"
	@echo "  make run            - Build and run the server"
	@echo "  make dev            - Run with hot reload (requires air)"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Docker - Full Stack:"
	@echo "  make docker-up      - Start all Docker services"
	@echo "  make docker-down    - Stop all Docker services"
	@echo "  make docker-logs    - View Docker logs"
	@echo ""
	@echo "Docker - AI Demo (~3GB):"
	@echo "  make demo-up        - Start AI demo (minimal setup)"
	@echo "  make demo-down      - Stop AI demo"
	@echo "  make demo-logs      - View AI demo logs"
	@echo ""
	@echo "Docker - Cleanup (oslobodi prostor):"
	@echo "  make docker-clean       - Obriši dangling images i cache"
	@echo "  make docker-clean-all   - Obriši sve nekorišćene images"
	@echo "  make docker-clean-nuclear - Obriši SVE osim volumes"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Lint code"
	@echo "  make check          - Run all checks (fmt, lint, test)"
	@echo ""
