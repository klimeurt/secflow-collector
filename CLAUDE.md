# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go microservice called **secflow-collector** that periodically scans a GitHub organization's repositories and publishes them to a NATS queue for further processing. It's a Kubernetes-ready service with cron-based scheduling.

## Development Commands

### Building and Testing
```bash
# Build binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code  
make fmt

# Lint code (requires golangci-lint)
make lint

# Full CI pipeline
make ci
```

### Local Development
```bash
# Start local NATS server
make dev-setup

# Run application locally
make run
# Or directly with go
go run main.go

# Stop development environment
make dev-teardown
```

### Docker Operations
```bash
# Build Docker image
make docker-build

# Push to registry
make docker-push
```

### Kubernetes/Helm Operations
```bash
# Lint Helm chart
make helm-lint

# Package Helm chart
make helm-package

# Install to local Kubernetes (requires GITHUB_ORG and GITHUB_TOKEN env vars)
make install-local

# Uninstall from local Kubernetes
make uninstall-local
```

## Architecture

The application consists of a single `main.go` file with the following key components:

- **Config**: Environment-based configuration management
- **Scanner**: Main service that handles GitHub API interactions and NATS publishing
- **Repository**: Data structure representing GitHub repositories for NATS messages

### Key Dependencies
- `github.com/google/go-github/v57` - GitHub API client
- `github.com/nats-io/nats.go` - NATS messaging
- `github.com/robfig/cron/v3` - Cron scheduling
- `golang.org/x/oauth2` - OAuth2 authentication

### Environment Variables
Required:
- `GITHUB_ORG` - GitHub organization name
- `GITHUB_TOKEN` - GitHub personal access token

Optional:
- `NATS_URL` (default: "nats://localhost:4222")
- `NATS_SUBJECT` (default: "github.repositories") 
- `CRON_SCHEDULE` (default: "0 0 * * 0" - weekly)
- `RUN_ON_STARTUP` (default: false)

## Testing

The project uses standard Go testing:
```bash
go test -v -race ./...
```

For integration tests (requires running NATS):
```bash
go test -tags=integration -v ./...
```

## Deployment

The service is designed for Kubernetes deployment via Helm:

1. **Helm Chart Location**: `helm/secflow-collector/`
2. **Default Values**: See `helm/secflow-collector/values.yaml`
3. **Security**: Runs as non-root user (UID 1000), supports read-only filesystem

### Helm Configuration
The chart expects:
- GitHub token stored as Kubernetes secret
- NATS server accessible in cluster
- Configurable cron schedule and startup behavior

## Repository Structure

- `main.go` - Single-file application with all logic
- `Makefile` - Development and deployment commands
- `Dockerfile` - Multi-stage build for minimal container
- `helm/secflow-collector/` - Kubernetes Helm chart
- `go.mod/go.sum` - Go module definitions

## Message Format

Repositories are published to NATS as JSON with this structure:
```json
{
  "name": "repository-name",
  "clone_url": "https://github.com/org/repo.git", 
  "ssh_url": "git@github.com:org/repo.git",
  "https_url": "https://github.com/org/repo.git",
  "created_at": "2023-01-01T00:00:00Z",
  "updated_at": "2023-12-01T00:00:00Z",
  "language": "Go",
  "topics": ["microservice", "kubernetes"]
}
```