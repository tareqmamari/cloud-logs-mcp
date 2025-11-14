.PHONY: help build test lint clean run install deps docker-build docker-run

# Variables
BINARY_NAME=logs-mcp-server
VERSION?=0.1.0
BUILD_DIR=./bin
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Multi-platform build complete"

# Development targets
run: ## Run the server locally
	$(GO) run main.go

dev: ## Run in development mode with hot reload
	@echo "Running in development mode..."
	@export LOG_LEVEL=debug LOG_FORMAT=console ENVIRONMENT=development && $(GO) run main.go

# Dependency management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Testing targets
test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	$(GO) test -v -race -short ./...

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	$(GO) test -v -race -run Integration ./...

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -v -bench=. -benchmem ./...

# Code quality targets
lint: ## Run linters
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

sec: ## Run security checks
	@echo "Running security checks..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec -quiet ./...

vuln: ## Check for vulnerabilities
	@echo "Checking for vulnerabilities..."
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

check: fmt vet lint sec vuln test ## Run all checks

# Installation targets
install: build ## Install the binary
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installed to /usr/local/bin/$(BINARY_NAME)"

uninstall: ## Uninstall the binary
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled"

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .
	@echo "Docker image built: $(BINARY_NAME):$(VERSION)"

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -it --env-file .env $(BINARY_NAME):latest

# Cleanup targets
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

clean-all: clean ## Clean all generated files
	@echo "Deep cleaning..."
	@$(GO) clean -cache -testcache -modcache
	@echo "Deep clean complete"

# Release targets
release: check build-all ## Create a release
	@echo "Creating release $(VERSION)..."
	@mkdir -p release
	@cd $(BUILD_DIR) && \
		for f in $(BINARY_NAME)-*; do \
			tar czf ../release/$$f-$(VERSION).tar.gz $$f; \
		done
	@echo "Release files created in ./release/"

# Development helpers
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@which fswatch > /dev/null || (echo "Please install fswatch" && exit 1)
	@fswatch -o . | xargs -n1 -I{} make build

.PHONY: proto
proto: ## Generate protobuf code (if needed)
	@echo "Generating protobuf code..."
	@# Add protobuf generation commands here if needed

# Environment setup
setup: deps ## Set up development environment
	@echo "Setting up development environment..."
	@cp -n .env.example .env || true
	@echo "Created .env file (if not exists)"
	@echo "Please edit .env and add your credentials"
	@echo "Setup complete"

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@which godoc > /dev/null || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	@echo "Starting godoc server on :6060"
	@echo "Visit http://localhost:6060/pkg/github.com/observability-c/logs-mcp-server/"
	godoc -http=:6060
