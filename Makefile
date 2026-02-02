.PHONY: build all clean test windows darwin-amd64 darwin-arm64

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

BINARY_NAME := broadcast-relay
BUILD_DIR := build

# Default target
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build all platforms
all: clean windows darwin-amd64 darwin-arm64

# Windows AMD64
windows:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# macOS Intel
darwin-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .

# macOS Apple Silicon
darwin-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe

# Install locally
install: build
	cp $(BINARY_NAME) /usr/local/bin/

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build for current platform"
	@echo "  all          - Build for all platforms (Windows, macOS Intel, macOS ARM)"
	@echo "  windows      - Build for Windows AMD64"
	@echo "  darwin-amd64 - Build for macOS Intel"
	@echo "  darwin-arm64 - Build for macOS Apple Silicon"
	@echo "  test         - Run tests"
	@echo "  clean        - Remove build artifacts"
	@echo "  install      - Install to /usr/local/bin"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
