# Gogo API Build Configuration

# Variables
APP_NAME=gogo-api
BUILD_DIR=build
SWAGGER_DIR=docs
MAIN_FILE=main.go

# Default target
.PHONY: all
all: deps swagger build

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Install swag if not present
.PHONY: install-swag
install-swag:
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)

# Generate Swagger documentation
.PHONY: swagger
swagger: install-swag
	@echo "Generating Swagger documentation..."
	swag init --output $(SWAGGER_DIR)

# Build the application
.PHONY: build
build: swagger
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Development mode with auto-rebuild
.PHONY: dev
dev: swagger
	@echo "Starting development server..."
	@echo "Swagger docs available at: http://localhost:3001/swagger/"
	go run $(MAIN_FILE)

# Test the application
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(SWAGGER_DIR)

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Run the built application
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# Generate docs and serve for development
.PHONY: docs-serve
docs-serve: swagger dev

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all        - Install deps, generate docs, and build"
	@echo "  deps       - Install Go dependencies"
	@echo "  swagger    - Generate Swagger documentation"
	@echo "  build      - Build the application"
	@echo "  dev        - Run in development mode with docs generation"
	@echo "  test       - Run tests"
	@echo "  clean      - Clean build artifacts and docs"
	@echo "  fmt        - Format Go code"
	@echo "  lint       - Lint Go code"
	@echo "  run        - Build and run the application"
	@echo "  help       - Show this help message"