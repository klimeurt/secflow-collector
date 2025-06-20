.PHONY: build build-collector build-validator test run run-collector run-validator docker-build docker-build-collector docker-build-validator docker-push docker-push-collector docker-push-validator clean

# Variables
COLLECTOR_NAME := secflow-collector
VALIDATOR_NAME := secflow-validator
DOCKER_REGISTRY := ghcr.io/klimeurt
COLLECTOR_IMAGE := $(DOCKER_REGISTRY)/$(COLLECTOR_NAME)
VALIDATOR_IMAGE := $(DOCKER_REGISTRY)/$(VALIDATOR_NAME)
VERSION := $(shell git describe --tags --always --dirty)

# Build binaries
build: build-collector build-validator

build-collector:
	go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(COLLECTOR_NAME) ./cmd/collector

build-validator:
	go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(VALIDATOR_NAME) ./cmd/validator

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
run: run-collector

run-collector:
	go run ./cmd/collector

run-validator:
	go run ./cmd/validator

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Docker build
docker-build: docker-build-collector docker-build-validator

docker-build-collector:
	docker build -f deployments/collector/Dockerfile -t $(COLLECTOR_IMAGE):$(VERSION) -t $(COLLECTOR_IMAGE):latest .

docker-build-validator:
	docker build -f deployments/validator/Dockerfile -t $(VALIDATOR_IMAGE):$(VERSION) -t $(VALIDATOR_IMAGE):latest .

# Docker push
docker-push: docker-push-collector docker-push-validator

docker-push-collector: docker-build-collector
	docker push $(COLLECTOR_IMAGE):$(VERSION)
	docker push $(COLLECTOR_IMAGE):latest

docker-push-validator: docker-build-validator
	docker push $(VALIDATOR_IMAGE):$(VERSION)
	docker push $(VALIDATOR_IMAGE):latest


# Install dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -f $(COLLECTOR_NAME) $(VALIDATOR_NAME)
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

