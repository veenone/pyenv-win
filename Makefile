# Makefile for pyenv-win-go
# Supports multiple build types, platforms, and configurations

# Shell configuration - Use PowerShell for Windows
SHELL := powershell.exe
.SHELLFLAGS := -NoProfile -Command

# Project information
PROJECT_NAME := pyenv
VERSION := 4.0.0
BUILD_DATE := $(shell Get-Date -Format "yyyy-MM-dd")
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>$$null; if ($$?) { } else { "unknown" })

# Directories
BUILD_DIR := build
SRC_DIR := pyenv-win-go
CMD_DIR := $(SRC_DIR)/cmd/pyenv
INTERNAL_DIR := $(SRC_DIR)/internal

# Output binaries
BINARY_STANDARD := $(BUILD_DIR)/pyenv.exe
BINARY_MIRROR := $(BUILD_DIR)/pyenv-mirror.exe
BINARY_DEBUG := $(BUILD_DIR)/pyenv-debug.exe

# Go build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.commitHash=$(COMMIT_HASH)"
LDFLAGS_PROD := -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.commitHash=$(COMMIT_HASH) -s -w"
BUILD_FLAGS := -trimpath
BUILD_FLAGS_DEBUG := -gcflags="all=-N -l"

# Go command
GO := go
GOFMT := gofmt
GOTEST := go test
GOVET := go vet

# Colors for output (disabled for Windows compatibility)
COLOR_RESET :=
COLOR_BOLD :=
COLOR_GREEN :=
COLOR_YELLOW :=
COLOR_BLUE :=
COLOR_RED :=

# Default target
.DEFAULT_GOAL := help

#########################################
# Help Target
#########################################

.PHONY: help
help: ## Show this help message
	@echo "$(COLOR_BOLD)pyenv-win-go Makefile$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Version: $(VERSION)$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Available targets:$(COLOR_RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(COLOR_BOLD)Build Types:$(COLOR_RESET)"
	@echo "  $(COLOR_YELLOW)standard$(COLOR_RESET)  - Core pyenv-win-go (no mirror feature)"
	@echo "  $(COLOR_YELLOW)mirror$(COLOR_RESET)    - With Nexus mirroring support"
	@echo "  $(COLOR_YELLOW)debug$(COLOR_RESET)     - With debug symbols"
	@echo "  $(COLOR_YELLOW)all$(COLOR_RESET)       - Build all variants"
	@echo ""
	@echo "$(COLOR_BOLD)Examples:$(COLOR_RESET)"
	@echo "  make build         # Build standard version"
	@echo "  make build-mirror  # Build with mirror feature"
	@echo "  make build-all     # Build all variants"
	@echo "  make install       # Install to GOPATH/bin"
	@echo "  make clean         # Clean build artifacts"

#########################################
# Build Targets
#########################################

.PHONY: build
build: standard ## Build standard version (default)

.PHONY: standard
standard: ## Build standard version without mirror feature
	@Write-Host "Building standard version..." -ForegroundColor Blue
	@New-Item -ItemType Directory -Force -Path $(BUILD_DIR) | Out-Null
	@Set-Location $(SRC_DIR); $(GO) build $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BINARY_STANDARD) ./cmd/pyenv
	@Write-Host "Standard build complete: $(BINARY_STANDARD)" -ForegroundColor Green

.PHONY: build-mirror
build-mirror: mirror ## Alias for mirror target

.PHONY: mirror
mirror: ## Build version with Nexus mirror feature
	@Write-Host "Building mirror version..." -ForegroundColor Blue
	@New-Item -ItemType Directory -Force -Path $(BUILD_DIR) | Out-Null
	@Set-Location $(SRC_DIR); $(GO) build -tags mirror $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BINARY_MIRROR) ./cmd/pyenv
	@Write-Host "Mirror build complete: $(BINARY_MIRROR)" -ForegroundColor Green

.PHONY: debug
debug: clean-build-dir ## Build version with debug symbols
	@echo "$(COLOR_BLUE)Building debug version...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	cd $(SRC_DIR) && $(GO) build -tags mirror $(BUILD_FLAGS_DEBUG) $(LDFLAGS) -o ../$(BINARY_DEBUG) ./cmd/pyenv
	@echo "$(COLOR_GREEN)✓ Debug build complete: $(BINARY_DEBUG)$(COLOR_RESET)"

.PHONY: build-all
build-all: standard mirror ## Build all variants (standard + mirror)
	@echo "$(COLOR_GREEN)✓ All builds complete$(COLOR_RESET)"

.PHONY: build-full
build-full: standard mirror debug ## Build all variants including debug
	@echo "$(COLOR_GREEN)✓ All builds complete (including debug)$(COLOR_RESET)"

#########################################
# Cross-Platform Builds
#########################################

.PHONY: build-windows
build-windows: ## Build for Windows (64-bit and 32-bit)
	@echo "$(COLOR_BLUE)Building for Windows...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)/windows
	cd $(SRC_DIR) && GOOS=windows GOARCH=amd64 $(GO) build $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/windows/pyenv-amd64.exe ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=windows GOARCH=amd64 $(GO) build -tags mirror $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/windows/pyenv-mirror-amd64.exe ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=windows GOARCH=386 $(GO) build $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/windows/pyenv-386.exe ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=windows GOARCH=386 $(GO) build -tags mirror $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/windows/pyenv-mirror-386.exe ./cmd/pyenv
	@echo "$(COLOR_GREEN)✓ Windows builds complete$(COLOR_RESET)"

.PHONY: build-linux
build-linux: ## Build for Linux (64-bit and 32-bit)
	@echo "$(COLOR_BLUE)Building for Linux...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)/linux
	cd $(SRC_DIR) && GOOS=linux GOARCH=amd64 $(GO) build $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/linux/pyenv-amd64 ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=linux GOARCH=amd64 $(GO) build -tags mirror $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/linux/pyenv-mirror-amd64 ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=linux GOARCH=386 $(GO) build $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/linux/pyenv-386 ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=linux GOARCH=386 $(GO) build -tags mirror $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/linux/pyenv-mirror-386 ./cmd/pyenv
	@echo "$(COLOR_GREEN)✓ Linux builds complete$(COLOR_RESET)"

.PHONY: build-darwin
build-darwin: ## Build for macOS (Intel and ARM64)
	@echo "$(COLOR_BLUE)Building for macOS...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)/darwin
	cd $(SRC_DIR) && GOOS=darwin GOARCH=amd64 $(GO) build $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/darwin/pyenv-amd64 ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=darwin GOARCH=amd64 $(GO) build -tags mirror $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/darwin/pyenv-mirror-amd64 ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=darwin GOARCH=arm64 $(GO) build $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/darwin/pyenv-arm64 ./cmd/pyenv
	cd $(SRC_DIR) && GOOS=darwin GOARCH=arm64 $(GO) build -tags mirror $(BUILD_FLAGS) $(LDFLAGS_PROD) -o ../$(BUILD_DIR)/darwin/pyenv-mirror-arm64 ./cmd/pyenv
	@echo "$(COLOR_GREEN)✓ macOS builds complete$(COLOR_RESET)"

.PHONY: build-cross-platform
build-cross-platform: build-windows build-linux build-darwin ## Build for all platforms
	@echo "$(COLOR_GREEN)✓ All cross-platform builds complete$(COLOR_RESET)"

#########################################
# Development Targets
#########################################

.PHONY: dev
dev: ## Build and test quickly (no optimization)
	@echo "$(COLOR_BLUE)Building development version...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GO) build -o ../$(BUILD_DIR)/pyenv-dev.exe ./cmd/pyenv
	@echo "$(COLOR_GREEN)✓ Development build complete$(COLOR_RESET)"

.PHONY: run
run: dev ## Build and run with arguments (make run ARGS="--version")
	@echo "$(COLOR_BLUE)Running pyenv-win-go...$(COLOR_RESET)"
	./$(BUILD_DIR)/pyenv-dev.exe $(ARGS)

.PHONY: install
install: standard ## Install standard version to GOPATH/bin
	@echo "$(COLOR_BLUE)Installing to GOPATH/bin...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GO) install $(BUILD_FLAGS) $(LDFLAGS_PROD) ./cmd/pyenv
	@echo "$(COLOR_GREEN)✓ Installed successfully$(COLOR_RESET)"
	@which pyenv || echo "$(COLOR_YELLOW)Note: Make sure GOPATH/bin is in your PATH$(COLOR_RESET)"

.PHONY: install-mirror
install-mirror: mirror ## Install mirror version to GOPATH/bin
	@echo "$(COLOR_BLUE)Installing mirror version to GOPATH/bin...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GO) install -tags mirror $(BUILD_FLAGS) $(LDFLAGS_PROD) ./cmd/pyenv
	@echo "$(COLOR_GREEN)✓ Mirror version installed successfully$(COLOR_RESET)"

#########################################
# Testing Targets
#########################################

.PHONY: test
test: ## Run all tests
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GOTEST) -v ./...
	@echo "$(COLOR_GREEN)✓ Tests passed$(COLOR_RESET)"

.PHONY: test-mirror
test-mirror: ## Run tests with mirror tag
	@echo "$(COLOR_BLUE)Running tests with mirror tag...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GOTEST) -tags mirror -v ./...
	@echo "$(COLOR_GREEN)✓ Mirror tests passed$(COLOR_RESET)"

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "$(COLOR_BLUE)Running tests with coverage...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)/coverage
	cd $(SRC_DIR) && $(GOTEST) -coverprofile=../$(BUILD_DIR)/coverage/coverage.out ./...
	cd $(SRC_DIR) && $(GO) tool cover -html=../$(BUILD_DIR)/coverage/coverage.out -o ../$(BUILD_DIR)/coverage/coverage.html
	@echo "$(COLOR_GREEN)✓ Coverage report generated: $(BUILD_DIR)/coverage/coverage.html$(COLOR_RESET)"

.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "$(COLOR_BLUE)Running tests with race detector...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GOTEST) -race -v ./...
	@echo "$(COLOR_GREEN)✓ Race detector tests passed$(COLOR_RESET)"

.PHONY: bench
bench: ## Run benchmarks
	@echo "$(COLOR_BLUE)Running benchmarks...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GOTEST) -bench=. -benchmem ./...

#########################################
# Code Quality Targets
#########################################

.PHONY: fmt
fmt: ## Format Go code
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GOFMT) -s -w .
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@echo "$(COLOR_BLUE)Checking code format...$(COLOR_RESET)"
	@cd $(SRC_DIR) && test -z "$$($(GOFMT) -s -l . | tee /dev/stderr)" || (echo "$(COLOR_RED)✗ Code is not formatted. Run 'make fmt'$(COLOR_RESET)" && exit 1)
	@echo "$(COLOR_GREEN)✓ Code is properly formatted$(COLOR_RESET)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GOVET) ./...
	@echo "$(COLOR_GREEN)✓ Vet passed$(COLOR_RESET)"

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint installed)
	@echo "$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@command -v golangci-lint >/dev/null 2>&1 || { echo "$(COLOR_RED)✗ golangci-lint not installed. Install from: https://golangci-lint.run/$(COLOR_RESET)"; exit 1; }
	cd $(SRC_DIR) && golangci-lint run ./...
	@echo "$(COLOR_GREEN)✓ Linting passed$(COLOR_RESET)"

.PHONY: check
check: fmt-check vet ## Run all code quality checks
	@echo "$(COLOR_GREEN)✓ All checks passed$(COLOR_RESET)"

#########################################
# Dependency Management
#########################################

.PHONY: deps
deps: ## Download dependencies
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GO) mod download
	@echo "$(COLOR_GREEN)✓ Dependencies downloaded$(COLOR_RESET)"

.PHONY: deps-tidy
deps-tidy: ## Tidy and verify dependencies
	@echo "$(COLOR_BLUE)Tidying dependencies...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GO) mod tidy
	cd $(SRC_DIR) && $(GO) mod verify
	@echo "$(COLOR_GREEN)✓ Dependencies tidied$(COLOR_RESET)"

.PHONY: deps-upgrade
deps-upgrade: ## Upgrade all dependencies
	@echo "$(COLOR_BLUE)Upgrading dependencies...$(COLOR_RESET)"
	cd $(SRC_DIR) && $(GO) get -u ./...
	cd $(SRC_DIR) && $(GO) mod tidy
	@echo "$(COLOR_GREEN)✓ Dependencies upgraded$(COLOR_RESET)"

#########################################
# Clean Targets
#########################################

.PHONY: clean
clean: clean-build clean-test ## Clean all build artifacts and test cache

.PHONY: clean-build
clean-build: ## Clean build artifacts
	@Write-Host "Cleaning build artifacts..." -ForegroundColor Blue
	@if (Test-Path $(BUILD_DIR)) { Remove-Item -Recurse -Force $(BUILD_DIR) }
	@Write-Host "Build artifacts cleaned" -ForegroundColor Green

.PHONY: clean-build-dir
clean-build-dir:
	if (-not (Test-Path $(BUILD_DIR))) { New-Item -ItemType Directory -Path $(BUILD_DIR) | Out-Null }

.PHONY: clean-test
clean-test: ## Clean test cache
	Set-Location $(SRC_DIR); $(GO) clean -testcache

#########################################
# Release Targets
#########################################

.PHONY: release
release: clean build-cross-platform ## Build release binaries for all platforms
	@echo "$(COLOR_BLUE)Creating release archives...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)/release
	# Windows
	cd $(BUILD_DIR)/windows && zip -q ../release/pyenv-win-$(VERSION)-amd64.zip pyenv-amd64.exe pyenv-mirror-amd64.exe
	cd $(BUILD_DIR)/windows && zip -q ../release/pyenv-win-$(VERSION)-386.zip pyenv-386.exe pyenv-mirror-386.exe
	# Linux
	cd $(BUILD_DIR)/linux && tar czf ../release/pyenv-linux-$(VERSION)-amd64.tar.gz pyenv-amd64 pyenv-mirror-amd64
	cd $(BUILD_DIR)/linux && tar czf ../release/pyenv-linux-$(VERSION)-386.tar.gz pyenv-386 pyenv-mirror-386
	# macOS
	cd $(BUILD_DIR)/darwin && tar czf ../release/pyenv-darwin-$(VERSION)-amd64.tar.gz pyenv-amd64 pyenv-mirror-amd64
	cd $(BUILD_DIR)/darwin && tar czf ../release/pyenv-darwin-$(VERSION)-arm64.tar.gz pyenv-arm64 pyenv-mirror-arm64
	@echo "$(COLOR_GREEN)✓ Release archives created in $(BUILD_DIR)/release/$(COLOR_RESET)"

.PHONY: checksums
checksums: ## Generate SHA256 checksums for release files
	@echo "$(COLOR_BLUE)Generating checksums...$(COLOR_RESET)"
	cd $(BUILD_DIR)/release && sha256sum * > SHA256SUMS.txt
	@echo "$(COLOR_GREEN)✓ Checksums generated: $(BUILD_DIR)/release/SHA256SUMS.txt$(COLOR_RESET)"
	@cat $(BUILD_DIR)/release/SHA256SUMS.txt

#########################################
# Information Targets
#########################################

.PHONY: version
version: ## Show version information
	@echo "$(COLOR_BOLD)pyenv-win-go$(COLOR_RESET)"
	@echo "Version:     $(VERSION)"
	@echo "Build Date:  $(BUILD_DATE)"
	@echo "Commit Hash: $(COMMIT_HASH)"

.PHONY: info
info: version ## Show project information
	@echo ""
	@echo "$(COLOR_BOLD)Build Directories:$(COLOR_RESET)"
	@echo "Source:      $(SRC_DIR)"
	@echo "Build:       $(BUILD_DIR)"
	@echo "Command:     $(CMD_DIR)"
	@echo ""
	@echo "$(COLOR_BOLD)Go Environment:$(COLOR_RESET)"
	@$(GO) version
	@$(GO) env GOOS GOARCH

.PHONY: list
list: ## List all available targets
	@$(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | grep -E -v -e '^[^[:alnum:]]' -e '^$@$$'

#########################################
# CI/CD Targets
#########################################

.PHONY: ci
ci: deps check test build-all ## Run CI pipeline (deps, checks, tests, build)
	@echo "$(COLOR_GREEN)✓ CI pipeline completed successfully$(COLOR_RESET)"

.PHONY: ci-full
ci-full: deps check test test-race build-cross-platform ## Run full CI pipeline with race tests
	@echo "$(COLOR_GREEN)✓ Full CI pipeline completed successfully$(COLOR_RESET)"

#########################################
# Docker Targets (Future)
#########################################

.PHONY: docker-build
docker-build: ## Build Docker image (requires Dockerfile)
	@echo "$(COLOR_YELLOW)Docker build not yet implemented$(COLOR_RESET)"

.PHONY: docker-run
docker-run: ## Run in Docker container
	@echo "$(COLOR_YELLOW)Docker run not yet implemented$(COLOR_RESET)"

#########################################
# Utility Targets
#########################################

.PHONY: size
size: ## Show binary sizes
	@echo "$(COLOR_BOLD)Binary Sizes:$(COLOR_RESET)"
	@if [ -f $(BINARY_STANDARD) ]; then du -h $(BINARY_STANDARD) | awk '{print "Standard: " $$1}'; fi
	@if [ -f $(BINARY_MIRROR) ]; then du -h $(BINARY_MIRROR) | awk '{print "Mirror:   " $$1}'; fi
	@if [ -f $(BINARY_DEBUG) ]; then du -h $(BINARY_DEBUG) | awk '{print "Debug:    " $$1}'; fi

.PHONY: watch
watch: ## Watch for changes and rebuild (requires entr)
	@command -v entr >/dev/null 2>&1 || { echo "$(COLOR_RED)✗ entr not installed. Install from: https://eradman.com/entrproject/$(COLOR_RESET)"; exit 1; }
	@echo "$(COLOR_BLUE)Watching for changes...$(COLOR_RESET)"
	@find $(SRC_DIR) -name '*.go' | entr -c make dev

.PHONY: tree
tree: ## Show project structure
	@command -v tree >/dev/null 2>&1 || { echo "$(COLOR_RED)✗ tree not installed$(COLOR_RESET)"; exit 1; }
	@tree -L 3 -I 'vendor|node_modules' $(SRC_DIR)

# Make sure build directory exists
$(BUILD_DIR):
	New-Item -ItemType Directory -Force -Path $(BUILD_DIR) | Out-Null
