.PHONY: build test run docker-build docker-push helm-lint helm-package clean

# Variables
APP_NAME := secflow-collector
DOCKER_REGISTRY := your-registry.io
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(APP_NAME)
VERSION := $(shell git describe --tags --always --dirty)
HELM_CHART := helm/secflow-collector

# Build binary
build:
	go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(APP_NAME) main.go

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
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

# Helm lint
helm-lint:
	helm lint $(HELM_CHART)

# Helm package
helm-package:
	helm package $(HELM_CHART)

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

# Full CI pipeline
ci: lint test build docker-build helm-lint

# Install to local Kubernetes
install-local:
	kubectl create secret generic github-token --from-literal=token=$(GITHUB_TOKEN) || true
	helm upgrade --install $(APP_NAME) $(HELM_CHART) \
		--set github.organization=$(GITHUB_ORG) \
		--set image.repository=$(DOCKER_IMAGE) \
		--set image.tag=$(VERSION)

# Uninstall from local Kubernetes
uninstall-local:
	helm uninstall $(APP_NAME)
	kubectl delete secret github-token || true