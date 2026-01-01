# Makefile for dash0-api-client-go

# Variables
OPENAPI_URL := https://api.eu-west-1.aws.dash0-dev.com/openapi.yaml
OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@1401fbe26ce7e128e9963786742490ff444e3795

.PHONY: all generate build test lint clean tidy help

# Default target
all: clean generate tidy fmt lint build test

# Generate code from OpenAPI spec
generate:
	@echo "Generating code from OpenAPI spec..."
	$(OAPI_CODEGEN) --config=oapi-codegen.yaml $(OPENAPI_URL)
	@echo "Post-processing generated code to resolve naming conflicts..."
	@sed -i.bak \
		-e 's/ClientOption/generatedClientOption/g' \
		-e 's/NewClient(/newGeneratedClient(/g' \
		-e 's/WithHTTPClient/withGeneratedHTTPClient/g' \
		-e 's/WithBaseURL/withGeneratedApiUrl/g' \
		generated.go && rm generated.go.bak

# Build the library
build:
	@echo "Building..."
	go build ./...

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

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
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy go.mod"
	@echo "  deps           - Download dependencies"
	@echo "  clean          - Remove generated files"
	@echo "  verify-spec    - Check if OpenAPI spec URL is accessible"
	@echo "  help           - Show this help message"
