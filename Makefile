# ccl - Claude Code Log viewer
BINARY_NAME := ccl

# Default target
.DEFAULT_GOAL := all

# Format code
.PHONY: fmt
fmt:
	@go fmt ./...

# Lint code
.PHONY: lint
lint:
	@go vet ./...

# Run tests
.PHONY: test
test:
	@go test -v ./...

# Build binary
.PHONY: build
build:
	@go build -o $(BINARY_NAME) .

# Run all targets in order
.PHONY: all
all: fmt lint test build
