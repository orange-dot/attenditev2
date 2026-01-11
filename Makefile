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
	@echo "Available commands:"
	@echo "  make build        - Build the platform binary"
	@echo "  make run          - Build and run the server"
	@echo "  make dev          - Run with hot reload (requires air)"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage- Run tests with coverage report"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-up    - Start Docker services"
	@echo "  make docker-down  - Stop Docker services"
	@echo "  make docker-logs  - View Docker logs"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo "  make check        - Run all checks (fmt, lint, test)"
	@echo ""
