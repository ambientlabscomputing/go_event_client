# Go Event Client Makefile
# 
# This Makefile provides convenient commands for building, testing, and running
# the Go Event Client library and examples.

.PHONY: help build test test-unit test-integration test-models test-oauth test-all clean lint format examples run-demo deps check-env install

# Default target
.DEFAULT_GOAL := help

# Variables
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOMOD := $(GO) mod
GOFMT := $(GO) fmt
GOLINT := golangci-lint

# Test timeout settings
UNIT_TIMEOUT := 30s
INTEGRATION_TIMEOUT := 120s
OAUTH_TIMEOUT := 30s

# Test verbosity (use V=1 for verbose output)
VERBOSE := $(if $(V),-v,)

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
WHITE := \033[0;37m
NC := \033[0m # No Color

##@ General

help: ## Display this help message
	@echo "$(CYAN)Go Event Client - Available Make Targets$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(YELLOW)<target>$(NC)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(BLUE)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(PURPLE)Examples:$(NC)"
	@echo "  make test              # Run all tests"
	@echo "  make test-unit V=1     # Run unit tests with verbose output"
	@echo "  make test-integration  # Run integration tests"
	@echo "  make run-demo          # Run the comprehensive demo"
	@echo "  make check-env         # Check environment configuration"
	@echo ""

check-env: ## Check if required environment variables are set
	@echo "$(BLUE)Checking environment configuration...$(NC)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)Warning: .env file not found. Copy .env.example to .env and configure it.$(NC)"; \
		if [ -f .env.example ]; then \
			echo "$(CYAN)Available example:$(NC)"; \
			echo "  cp .env.example .env"; \
		fi; \
	else \
		echo "$(GREEN)✓ .env file found$(NC)"; \
	fi
	@echo "$(GREEN)Environment check complete$(NC)"

##@ Dependencies

deps: ## Download and verify dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) verify
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

install: deps ## Install the library (same as deps for libraries)
	@echo "$(GREEN)✓ Library ready for use$(NC)"

##@ Build

build: ## Build the library and examples
	@echo "$(BLUE)Building library...$(NC)"
	$(GOBUILD) ./...
	@echo "$(BLUE)Building examples...$(NC)"
	cd examples && $(GOBUILD) -o comprehensive_demo comprehensive_demo.go
	@echo "$(BLUE)Building samples...$(NC)"
	cd samples && $(GOBUILD) -o samples sample.go
	@echo "$(GREEN)✓ Build complete$(NC)"

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	$(GO) clean ./...
	rm -f examples/comprehensive_demo
	rm -f samples/samples
	@echo "$(GREEN)✓ Clean complete$(NC)"

##@ Code Quality

format: ## Format Go code
	@echo "$(BLUE)Formatting code...$(NC)"
	$(GOFMT) ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

lint: ## Run linter (requires golangci-lint)
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
		echo "$(GREEN)✓ Linting complete$(NC)"; \
	else \
		echo "$(YELLOW)Warning: golangci-lint not installed. Install with:$(NC)"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

##@ Testing

test: test-unit test-integration ## Run all tests
	@echo "$(GREEN)✓ All tests completed$(NC)"

test-all: deps test lint ## Run all tests and linting
	@echo "$(GREEN)✓ Full test suite completed$(NC)"

test-unit: ## Run unit tests only
	@echo "$(BLUE)Running unit tests...$(NC)"
	$(GOTEST) $(VERBOSE) -timeout $(UNIT_TIMEOUT) -run "^Test[^I]" ./...
	@echo "$(GREEN)✓ Unit tests completed$(NC)"

test-models: ## Run model/structure tests
	@echo "$(BLUE)Running model tests...$(NC)"
	$(GOTEST) $(VERBOSE) -timeout $(UNIT_TIMEOUT) -run "TestMessage_|TestSubscription_|TestSubscriber_|TestSession_" ./...
	@echo "$(GREEN)✓ Model tests completed$(NC)"

test-integration: check-env ## Run integration tests (requires .env configuration)
	@echo "$(BLUE)Running integration tests...$(NC)"
	@echo "$(YELLOW)Note: Integration tests require valid .env configuration$(NC)"
	$(GOTEST) $(VERBOSE) -timeout $(INTEGRATION_TIMEOUT) -run "TestIntegration_" ./...
	@echo "$(GREEN)✓ Integration tests completed$(NC)"

test-oauth: check-env ## Run OAuth authentication tests
	@echo "$(BLUE)Running OAuth tests...$(NC)"
	$(GOTEST) $(VERBOSE) -timeout $(OAUTH_TIMEOUT) -run "TestOAuth" ./...
	@echo "$(GREEN)✓ OAuth tests completed$(NC)"

test-basic: ## Run basic subscription test
	@echo "$(BLUE)Running basic subscription test...$(NC)"
	$(GOTEST) $(VERBOSE) -timeout $(INTEGRATION_TIMEOUT) -run "TestIntegration_BasicSubscription" ./...

test-aggregate: ## Run aggregate subscription tests  
	@echo "$(BLUE)Running aggregate subscription tests...$(NC)"
	$(GOTEST) $(VERBOSE) -timeout $(INTEGRATION_TIMEOUT) -run "TestIntegration_.*Aggregate" ./...

test-regex: ## Run regex subscription test
	@echo "$(BLUE)Running regex subscription test...$(NC)"
	$(GOTEST) $(VERBOSE) -timeout $(INTEGRATION_TIMEOUT) -run "TestIntegration_RegexSubscription" ./...

test-short: ## Run tests in short mode (skips integration tests)
	@echo "$(BLUE)Running tests in short mode...$(NC)"
	$(GOTEST) $(VERBOSE) -short -timeout $(UNIT_TIMEOUT) ./...
	@echo "$(GREEN)✓ Short tests completed$(NC)"

test-verbose: ## Run all tests with verbose output
	@echo "$(BLUE)Running all tests with verbose output...$(NC)"
	$(GOTEST) -v -timeout $(INTEGRATION_TIMEOUT) ./...

test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GOTEST) -v -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"

##@ Examples and Demos

examples: build ## Build all examples
	@echo "$(GREEN)✓ Examples built$(NC)"

run-demo: check-env examples ## Run the comprehensive demo
	@echo "$(BLUE)Running comprehensive demo...$(NC)"
	@echo "$(YELLOW)Make sure your .env file is configured with valid credentials$(NC)"
	set -a && . ./.env && set +a && cd examples && ./comprehensive_demo

run-sample: build ## Run the sample application
	@echo "$(BLUE)Running sample application...$(NC)"
	cd samples && ./samples

##@ Development

dev-setup: deps ## Set up development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(YELLOW)Created .env file from .env.example - please configure it$(NC)"; \
	fi
	@echo "$(CYAN)Installing development tools...$(NC)"
	@if ! command -v $(GOLINT) >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "$(GREEN)✓ Development environment ready$(NC)"

watch-test: ## Watch for file changes and run tests
	@echo "$(BLUE)Watching for changes... (Press Ctrl+C to stop)$(NC)"
	@while true; do \
		inotifywait -r -e modify --include='\.go$$' . 2>/dev/null && \
		clear && \
		echo "$(YELLOW)Files changed, running tests...$(NC)" && \
		make test-short; \
	done

##@ Maintenance

tidy: ## Tidy up go.mod and go.sum
	@echo "$(BLUE)Tidying Go modules...$(NC)"
	$(GOMOD) tidy
	@echo "$(GREEN)✓ Modules tidied$(NC)"

update: ## Update dependencies to latest versions
	@echo "$(BLUE)Updating dependencies...$(NC)"
	$(GO) get -u ./...
	$(GOMOD) tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

verify: tidy test-all ## Verify the project (tidy + test + lint)
	@echo "$(GREEN)✓ Project verification complete$(NC)"

##@ CI/CD

ci: deps test-all ## Run CI pipeline (dependencies + tests + linting)
	@echo "$(GREEN)✓ CI pipeline completed$(NC)"

pre-commit: format lint test-short ## Run pre-commit checks
	@echo "$(GREEN)✓ Pre-commit checks completed$(NC)"

##@ Information

info: ## Display project information
	@echo "$(CYAN)Go Event Client Project Information$(NC)"
	@echo ""
	@echo "$(BLUE)Go Version:$(NC)     $$($(GO) version)"
	@echo "$(BLUE)Module:$(NC)         $$($(GO) list -m)"
	@echo "$(BLUE)Dependencies:$(NC)   $$($(GO) list -m all | wc -l) total"
	@echo "$(BLUE)Test Files:$(NC)     $$(find . -name '*_test.go' | wc -l) files"
	@echo "$(BLUE)Source Files:$(NC)   $$(find . -name '*.go' -not -name '*_test.go' | wc -l) files"
	@echo ""
	@if [ -f .env ]; then \
		echo "$(GREEN)✓ Environment configured$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Environment not configured$(NC)"; \
	fi

version: ## Display version information
	@echo "$(CYAN)Version Information$(NC)"
	@$(GO) version
	@echo "Module: $$($(GO) list -m)"

##@ Help

list: ## List all available targets
	@echo "$(CYAN)Available Make Targets:$(NC)"
	@$(MAKE) -qp | awk -F':' '/^[a-zA-Z0-9][^$$#\/\t=]*:([^=]|$$)/ {split($$1,A,/ /);for(i in A)print A[i]}' | sort | uniq

# Add some useful aliases
t: test ## Alias for test
ut: test-unit ## Alias for test-unit  
it: test-integration ## Alias for test-integration
b: build ## Alias for build
c: clean ## Alias for clean
h: help ## Alias for help
