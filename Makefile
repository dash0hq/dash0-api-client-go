# Makefile for dash0-api-client-go

# Variables
OPENAPI_URL := https://api.eu-west-1.aws.dash0-dev.com/openapi.yaml
OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@1401fbe26ce7e128e9963786742490ff444e3795

.PHONY: all generate build test test-collector lint clean tidy api-compat help

# Default target
all: clean generate tidy fmt lint build test

# Generate code from OpenAPI spec
generate:
	@echo "Generating code from OpenAPI spec..."
	$(OAPI_CODEGEN) --config=oapi-codegen.yaml $(OPENAPI_URL)
	@echo "Post-processing generated code..."
	@go run ./tools/postprocess generated.go

# Build the library
build:
	@echo "Building..."
	go build ./...

# Run tests (excludes collector integration test)
test:
	@echo "Running tests..."
	go test -v -race -cover -skip TestOTLPCollector ./...

# Run OTel Collector integration test
test-collector:
	@echo "Running OTel Collector integration test..."
	go test -v -race -run TestOTLPCollector ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	@echo "Running linter..."
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2 run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Check API compatibility against latest release tag.
# gorelease detects all incompatible changes; the script then filters out
# changes listed in api_compatibility_exceptions.txt.
# If unallowed incompatible changes remain, the target fails.
api-compat:
	@echo "Checking API compatibility..."
	@go run golang.org/x/exp/cmd/gorelease@latest 2>&1 | tee /tmp/gorelease-output.txt; \
	INCOMPAT=$$(sed -n '/^## incompatible changes/,/^## compatible changes/p' /tmp/gorelease-output.txt | grep -v '^##' | grep -v '^$$' | grep -v ': added$$'); \
	if [ -z "$$INCOMPAT" ]; then \
		echo "No incompatible changes detected."; \
		exit 0; \
	fi; \
	if [ -f api_compatibility_exceptions.txt ]; then \
		PATTERNS=$$(grep -v '^\s*#' api_compatibility_exceptions.txt | grep -v '^\s*$$' | sed 's/\./\\./g; s/\*/.*/g' | sed 's/^/^/; s/$$$$/$$$$/' | paste -sd'|' -); \
		UNALLOWED=$$(echo "$$INCOMPAT" | while IFS= read -r line; do \
			SYMBOL=$$(echo "$$line" | sed 's/:.*//' | tr -d ' '); \
			if ! echo "$$SYMBOL" | grep -qE "$$PATTERNS"; then \
				echo "  $$line"; \
			fi; \
		done); \
	else \
		UNALLOWED=$$(echo "$$INCOMPAT" | sed 's/^/  /'); \
	fi; \
	if [ -n "$$UNALLOWED" ]; then \
		echo ""; \
		echo "Unallowed incompatible changes:"; \
		echo "$$UNALLOWED"; \
		echo ""; \
		echo "If intentional, add the symbol name to api_compatibility_exceptions.txt."; \
		exit 1; \
	else \
		echo "All incompatible changes are allowed via api_compatibility_exceptions.txt."; \
	fi

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

# Clean generated files
clean:
	@echo "Cleaning..."
	rm -f generated.go
	rm -f coverage.out coverage.html

# Verify OpenAPI spec is accessible
verify-spec:
	@echo "Verifying OpenAPI spec..."
	@curl -sf $(OPENAPI_URL) > /dev/null && echo "OpenAPI spec is accessible" || echo "Failed to access OpenAPI spec"

# Help
help:
	@echo "Available targets:"
	@echo "  all            - Generate, build, and test (default)"
	@echo "  generate       - Generate code from OpenAPI spec"
	@echo "  build          - Build the library"
	@echo "  test           - Run tests (excludes collector integration test)"
	@echo "  test-collector - Run OTel Collector integration test"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy go.mod"
	@echo "  api-compat     - Check API compatibility against latest tag"
	@echo "  deps           - Download dependencies"
	@echo "  clean          - Remove generated files"
	@echo "  verify-spec    - Check if OpenAPI spec URL is accessible"
	@echo "  help           - Show this help message"
