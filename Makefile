.PHONY: build test run docker-build docker-push clean

# Variables
APP_NAME := secflow-collector
DOCKER_REGISTRY := ghcr.io/klimeurt
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(APP_NAME)
VERSION := $(shell git describe --tags --always --dirty)

# Build binary
build:
	go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(APP_NAME) main.go

# Run unit tests
test:
	go test -v -race ./...

# Run unit tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run integration tests (requires NATS server)
test-integration:
	go test -v -race -tags=integration ./...

# Run all tests (unit + integration)
test-all: test test-integration

# Run tests with coverage including integration tests
test-coverage-all:
	go test -v -race -tags=integration -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run locally
run:
	go run main.go

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Docker build
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

# Docker push
docker-push: docker-build
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest


# Install dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -f $(APP_NAME)
	rm -f coverage.out coverage.html
	rm -f *.tgz

# Development setup with NATS
dev-setup:
	docker run -d --name nats -p 4222:4222 nats:latest

# Stop development environment
dev-teardown:
	docker stop nats || true
	docker rm nats || true

# Full CI pipeline (unit tests only)
ci: lint test build docker-build

# Full CI pipeline with integration tests
ci-integration: lint test-all build docker-build

