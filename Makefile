.PHONY: help build test lint clean run install deps docker-build docker-run

# Variables
BINARY_NAME=logs-mcp-server
# Get version from git tags using svu (fallback to v0.0.0 if no tags exist)
VERSION?=$(shell svu current 2>/dev/null || echo "v0.0.0")
BUILD_DIR=./bin
GO=go
GOFLAGS=-v
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main._date=$(BUILD_DATE) -X main.builtBy=make"

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

test-integration: ## Run integration tests (requires IBM Cloud credentials)
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/...

test-integration-alerts: ## Run alert integration tests only
	@echo "Running alert integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/ -run TestAlerts

test-integration-policies: ## Run policy integration tests only
	@echo "Running policy integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/ -run TestPolicies

test-integration-e2m: ## Run E2M integration tests only
	@echo "Running E2M integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/ -run TestE2M

test-integration-views: ## Run view integration tests only
	@echo "Running view integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/ -run TestViews

test-integration-webhooks: ## Run webhook integration tests only
	@echo "Running webhook integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/ -run TestWebhooks

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -v -bench=. -benchmem ./...

# Code quality targets
lint: ## Run linters
	@echo "Running linters..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@$(shell go env GOPATH)/bin/golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@if ! command -v goimports &> /dev/null; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@$(shell go env GOPATH)/bin/goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

sec: ## Run security checks
	@echo "Running security checks..."
	@if ! command -v gosec &> /dev/null; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@$(shell go env GOPATH)/bin/gosec -exclude=G104,G304 -quiet ./...

vuln: ## Check for vulnerabilities
	@echo "Checking for vulnerabilities..."
	@if ! command -v govulncheck &> /dev/null; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@$(shell go env GOPATH)/bin/govulncheck ./...

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
release: ## Create a release with GoReleaser (dry-run)
	@echo "Running GoReleaser in snapshot mode..."
	@if ! command -v goreleaser &> /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@$(shell go env GOPATH)/bin/goreleaser release --snapshot --clean

release-dry-run: ## Test release without publishing
	@echo "Running GoReleaser dry-run..."
	@if ! command -v goreleaser &> /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@$(shell go env GOPATH)/bin/goreleaser release --skip=publish --clean

release-publish: ## Create and publish a release (requires git tag)
	@echo "Publishing release with GoReleaser..."
	@if ! command -v goreleaser &> /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@which gh > /dev/null || (echo "Error: GitHub CLI (gh) not found. Please install it: brew install gh" && exit 1)
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Exporting GitHub token..."; \
		export GITHUB_TOKEN=$$(gh auth token) && $(shell go env GOPATH)/bin/goreleaser release --clean; \
	else \
		echo "Using existing GITHUB_TOKEN"; \
		$(shell go env GOPATH)/bin/goreleaser release --clean; \
	fi

release-check: ## Validate GoReleaser configuration
	@echo "Validating GoReleaser configuration..."
	@if ! command -v goreleaser &> /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@$(shell go env GOPATH)/bin/goreleaser check

# Semantic versioning with svu
version-current: ## Show current version
	@echo "Current version:"
	@git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"

version-next: ## Calculate next version based on commits
	@echo "Next version:"
	@svu next

version-major: ## Calculate next major version
	@svu major

version-minor: ## Calculate next minor version
	@svu minor

version-patch: ## Calculate next patch version
	@svu patch

# Changelog management
changelog: ## Generate changelog from git history
	@echo "Generating changelog..."
	@export PATH="$$PATH:$$(go env GOPATH)/bin"; \
	which git-chglog > /dev/null || (echo "Installing git-chglog..." && go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest); \
	git-chglog -o CHANGELOG.md
	@echo "Changelog updated: CHANGELOG.md"

changelog-next: ## Preview changelog for next version
	@echo "Preview changelog for next version:"
	@export PATH="$$PATH:$$(go env GOPATH)/bin"; \
	which git-chglog > /dev/null || (echo "Installing git-chglog..." && go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest); \
	NEXT_VERSION=$$(svu next 2>/dev/null || echo "next"); \
	git-chglog --next-tag $$NEXT_VERSION $$NEXT_VERSION

version-tag: ## Create and push next version tag (automated release)
	@echo "Calculating next version..."
	@NEXT_VERSION=$$(svu next); \
	CURRENT_VERSION=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	if [ "$$NEXT_VERSION" = "$$CURRENT_VERSION" ]; then \
		echo "⚠️  No conventional commits found since $$CURRENT_VERSION"; \
		echo ""; \
		echo "For dependency updates, use: make release-patch"; \
		echo "For features, use 'feat:' prefix"; \
		echo "For bug fixes, use 'fix:' prefix"; \
		exit 1; \
	fi; \
	echo "Creating tag $$NEXT_VERSION..."; \
	git tag -a $$NEXT_VERSION -m "Release $$NEXT_VERSION"; \
	echo "Pushing tag to origin..."; \
	git push origin $$NEXT_VERSION; \
	echo "✅ Tag $$NEXT_VERSION created and pushed!"; \
	echo "GoReleaser will generate changelog in GitHub Release."

release-patch: ## Create patch release (for dependency updates, security fixes)
	@echo "Creating patch release..."
	@NEXT_VERSION=$$(svu patch); \
	CURRENT_VERSION=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	echo "Current version: $$CURRENT_VERSION"; \
	echo "Next version: $$NEXT_VERSION"; \
	echo ""; \
	echo "Creating tag $$NEXT_VERSION..."; \
	git tag -a $$NEXT_VERSION -m "Release $$NEXT_VERSION"; \
	echo "Pushing tag to origin..."; \
	git push origin $$NEXT_VERSION; \
	echo "✅ Tag $$NEXT_VERSION created and pushed!"; \
	echo "GoReleaser will generate changelog in GitHub Release."

release-minor: ## Create minor release (for new features)
	@echo "Creating minor release..."
	@NEXT_VERSION=$$(svu minor); \
	CURRENT_VERSION=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	echo "Current version: $$CURRENT_VERSION"; \
	echo "Next version: $$NEXT_VERSION"; \
	echo ""; \
	echo "Creating tag $$NEXT_VERSION..."; \
	git tag -a $$NEXT_VERSION -m "Release $$NEXT_VERSION"; \
	echo "Pushing tag to origin..."; \
	git push origin $$NEXT_VERSION; \
	echo "✅ Tag $$NEXT_VERSION created and pushed!"; \
	echo "GoReleaser will generate changelog in GitHub Release."

release-major: ## Create major release (for breaking changes)
	@echo "Creating major release..."
	@NEXT_VERSION=$$(svu major); \
	CURRENT_VERSION=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	echo "Current version: $$CURRENT_VERSION"; \
	echo "Next version: $$NEXT_VERSION"; \
	echo ""; \
	read -p "⚠️  This is a BREAKING CHANGE release. Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Aborted."; \
		exit 1; \
	fi; \
	echo "Creating tag $$NEXT_VERSION..."; \
	git tag -a $$NEXT_VERSION -m "Release $$NEXT_VERSION"; \
	echo "Pushing tag to origin..."; \
	git push origin $$NEXT_VERSION; \
	echo "✅ Tag $$NEXT_VERSION created and pushed!"; \
	echo "GoReleaser will generate changelog in GitHub Release."

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
	@if ! command -v godoc &> /dev/null; then \
		echo "Installing godoc..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
	fi
	@echo "Starting godoc server on :6060"
	@echo "Visit http://localhost:6060/pkg/github.com/tareqmamari/logs-mcp-server/"
	@$(shell go env GOPATH)/bin/godoc -http=:6060

# API Update helpers
compare-api: ## Compare old and new API definitions
	@./scripts/compare-api-changes.sh

backup-api: ## Backup current API definition
	@echo "Backing up API definition..."
	@cp logs-service-api.json logs-service-api.json.backup.$(shell date +%Y%m%d-%H%M%S)
	@echo "Backup created"

list-operations: ## List all operations in current API
	@echo "Operations in current API:"
	@grep -o '"operationId": "[^"]*"' logs-service-api.json | sed 's/"operationId": "\([^"]*\)"/\1/' | sort
