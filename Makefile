.PHONY: build run dev test clean migrate

# Build the application
build:
	@echo "Building..."
	@go build -o bin/bot cmd/bot/main.go

# Run the application
run: build
	@echo "Running..."
	@./bin/bot

# Run in development mode
dev:
	@echo "Running in development mode..."
	@go run cmd/bot/main.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Create .env from example
env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env file created from .env.example"; \
	else \
		echo ".env file already exists"; \
	fi

# Help
help:
	@echo "Available commands:"
	@echo "  make build   - Build the application"
	@echo "  make run     - Build and run the application"
	@echo "  make dev     - Run in development mode"
	@echo "  make test    - Run tests"
	@echo "  make clean   - Clean build artifacts"
	@echo "  make deps    - Install dependencies"
	@echo "  make fmt     - Format code"
	@echo "  make env     - Create .env from .env.example"
