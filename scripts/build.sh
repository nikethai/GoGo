#!/bin/bash

# Gogo API Build Script
# Automated build with Swagger documentation generation

set -e

APP_NAME="gogo-api"
BUILD_DIR="build"
DOCS_DIR="docs"

echo "🚀 Starting Gogo API build process..."

# Check if swag is installed
if ! command -v swag &> /dev/null; then
    echo "📦 Installing swag CLI tool..."
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# Install/update dependencies
echo "📦 Installing dependencies..."
go mod tidy
go mod download

# Generate Swagger documentation
echo "📚 Generating Swagger documentation..."
swag init --output $DOCS_DIR

# Create build directory
mkdir -p $BUILD_DIR

# Build the application
echo "🔨 Building $APP_NAME..."
go build -o $BUILD_DIR/$APP_NAME main.go

echo "✅ Build completed successfully!"
echo ""
echo "📋 Build Summary:"
echo "   - Executable: $BUILD_DIR/$APP_NAME"
echo "   - Swagger JSON: $DOCS_DIR/swagger.json"
echo "   - Swagger YAML: $DOCS_DIR/swagger.yaml"
echo ""
echo "🌐 To start the server:"
echo "   ./$BUILD_DIR/$APP_NAME"
echo ""
echo "📖 Once running, access:"
echo "   - API: http://localhost:3001"
echo "   - Swagger UI: http://localhost:3001/swagger/"
echo "   - Postman Import: http://localhost:3001/swagger/doc.json"