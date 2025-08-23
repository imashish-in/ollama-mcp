# MCPHost Makefile
# Common commands for development and deployment

.PHONY: help build test clean install docker-build docker-run docker-push release install-script

# Variables
BINARY_NAME=mcphost
VERSION?=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Default target
help: ## Show this help message
	@echo "MCPHost - AI-Powered Artifactory Management Tools"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
build: ## Build the application
	@echo "Building ${BINARY_NAME}..."
	go build ${LDFLAGS} -o ${BINARY_NAME} .

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe .

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f ${BINARY_NAME}
	rm -f ${BINARY_NAME}-*
	rm -f coverage.out coverage.html

# Installation commands
install: build ## Build and install locally
	@echo "Installing ${BINARY_NAME}..."
	cp ${BINARY_NAME} /usr/local/bin/
	@echo "Installation complete!"

install-local: build ## Build and install to ~/.local/bin
	@echo "Installing ${BINARY_NAME} to ~/.local/bin..."
	mkdir -p ~/.local/bin
	cp ${BINARY_NAME} ~/.local/bin/
	@echo "Installation complete! Add ~/.local/bin to your PATH if not already there."

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t your-username/${BINARY_NAME}:${VERSION} .
	docker tag your-username/${BINARY_NAME}:${VERSION} your-username/${BINARY_NAME}:latest

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -it --rm your-username/${BINARY_NAME}:latest

docker-push: docker-build ## Build and push Docker image
	@echo "Pushing Docker image..."
	docker push your-username/${BINARY_NAME}:${VERSION}
	docker push your-username/${BINARY_NAME}:latest

# Docker Compose commands
compose-up: ## Start all services with Docker Compose
	@echo "Starting services with Docker Compose..."
	docker-compose --profile full up -d

compose-down: ## Stop all services
	@echo "Stopping services..."
	docker-compose down

compose-logs: ## Show logs from all services
	@echo "Showing logs..."
	docker-compose logs -f

compose-artifactory: ## Start only Artifactory
	@echo "Starting Artifactory..."
	docker-compose --profile artifactory-only up -d artifactory

compose-ollama: ## Start only Ollama
	@echo "Starting Ollama..."
	docker-compose --profile ollama-only up -d ollama

# Release commands
release: build-all ## Prepare release binaries
	@echo "Preparing release ${VERSION}..."
	@mkdir -p releases/${VERSION}
	@cp ${BINARY_NAME}-* releases/${VERSION}/
	@cp scripts/install.sh releases/${VERSION}/
	@cp local.json releases/${VERSION}/config.json.example
	@echo "Release prepared in releases/${VERSION}/"

release-tag: ## Create and push a new release tag
	@echo "Creating release tag ${VERSION}..."
	git tag ${VERSION}
	git push origin ${VERSION}

# Documentation commands
docs-serve: ## Serve documentation locally
	@echo "Serving documentation..."
	@if command -v python3 >/dev/null 2>&1; then \
		python3 -m http.server 8000; \
	elif command -v python >/dev/null 2>&1; then \
		python -m SimpleHTTPServer 8000; \
	else \
		echo "Python not found. Please install Python to serve documentation."; \
	fi

docs-build: ## Build documentation
	@echo "Building documentation..."
	@mkdir -p docs/build
	@cp ARTIFACTORY_GUIDE.md docs/build/
	@cp ARTIFACTORY_QUICK_REFERENCE.md docs/build/
	@cp PUBLISHING_GUIDE.md docs/build/

# Setup commands
setup-dev: ## Setup development environment
	@echo "Setting up development environment..."
	go mod download
	go mod tidy
	@echo "Development environment ready!"

setup-docker: ## Setup Docker environment
	@echo "Setting up Docker environment..."
	docker-compose --profile artifactory-only up -d artifactory
	docker-compose --profile ollama-only up -d ollama
	@echo "Docker environment ready!"

# Utility commands
version: ## Show version information
	@echo "Version: ${VERSION}"
	@echo "Build Time: ${BUILD_TIME}"
	@echo "Git Commit: ${GIT_COMMIT}"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Security commands
security-scan: ## Run security scan
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Performance commands
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. ./...

# CI/CD commands
ci: test lint security-scan ## Run CI pipeline
	@echo "CI pipeline completed successfully!"

# Install script commands
install-script: ## Make install script executable
	@echo "Making install script executable..."
	chmod +x scripts/install.sh

# Quick start commands
quick-start: setup-docker ## Quick start with Docker
	@echo "Quick start completed!"
	@echo "Artifactory: http://localhost:8081 (admin/password)"
	@echo "Ollama: http://localhost:11434"
	@echo "Run: make compose-up to start MCPHost"

# Development workflow
dev: setup-dev build test ## Development workflow
	@echo "Development workflow completed!"

# Production build
prod: clean build-all test lint security-scan ## Production build
	@echo "Production build completed!"

# Help for specific commands
help-build: ## Show build help
	@echo "Build commands:"
	@echo "  make build        - Build for current platform"
	@echo "  make build-all    - Build for all platforms"
	@echo "  make install      - Install to /usr/local/bin"
	@echo "  make install-local - Install to ~/.local/bin"

help-docker: ## Show Docker help
	@echo "Docker commands:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"
	@echo "  make docker-push  - Push Docker image"
	@echo "  make compose-up   - Start all services"
	@echo "  make compose-down - Stop all services"

help-release: ## Show release help
	@echo "Release commands:"
	@echo "  make release      - Prepare release binaries"
	@echo "  make release-tag  - Create and push release tag"
	@echo "  make version      - Show version information"
