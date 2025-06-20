# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go microservice project containing two services:

- **secflow-collector**: Periodically scans a GitHub organization's repositories and publishes them to a NATS queue
- **secflow-validator**: Consumes repository messages, validates if repositories contain an `appsec-config.yml` file, and routes them to appropriate queues

## Development Commands

### Building and Testing
```bash
# Build all binaries
make build

# Build collector only
make build-collector

# Build validator only
make build-validator

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

# Run collector locally
make run-collector
# Or directly with go
go run ./cmd/collector

# Run validator locally
make run-validator
# Or directly with go
go run ./cmd/validator

# Stop development environment
make dev-teardown
```

### Docker Operations
```bash
# Build all Docker images
make docker-build

# Build collector Docker image
make docker-build-collector

# Build validator Docker image
make docker-build-validator

# Push all images to registry
make docker-push

# Push collector image only
make docker-push-collector

# Push validator image only
make docker-push-validator
```


## Architecture

The project consists of two main services:

### Collector Service
- **Config**: Environment-based configuration management
- **Scanner**: Main service that handles GitHub API interactions and NATS publishing
- **Repository**: Data structure representing GitHub repositories for NATS messages

### Validator Service
- **Validator**: Main service that subscribes to NATS messages and orchestrates validation
- **Checker**: GitHub API client that checks for `appsec-config.yml` file existence
- **Processor**: Handles message processing and routing based on validation results

### Key Dependencies
- `github.com/google/go-github/v57` - GitHub API client
- `github.com/nats-io/nats.go` - NATS messaging
- `github.com/robfig/cron/v3` - Cron scheduling
- `golang.org/x/oauth2` - OAuth2 authentication

### Environment Variables

#### Common (both services)
Required:
- `GITHUB_ORG` - GitHub organization name
- `GITHUB_TOKEN` - GitHub personal access token

Optional:
- `NATS_URL` (default: "nats://localhost:4222")

#### Collector-specific
Optional:
- `NATS_SUBJECT` (default: "github.repositories") 
- `CRON_SCHEDULE` (default: "0 0 * * 0" - weekly)
- `RUN_ON_STARTUP` (default: false)

#### Validator-specific
Optional:
- `SOURCE_SUBJECT` (default: "github.repositories") - Queue to consume from
- `VALID_REPOS_SUBJECT` (default: "repos.valid") - Queue for valid repositories
- `INVALID_REPOS_SUBJECT` (default: "repos.invalid") - Queue for invalid repositories
- `PROCESS_STARTUP_MESSAGES` (default: true) - Process existing messages in queue at startup

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

Both services are containerized and can be deployed to any container orchestration platform. The Docker images run as non-root user (UID 1000) and support read-only filesystem.

- Collector image: `ghcr.io/klimeurt/secflow-collector`
- Validator image: `ghcr.io/klimeurt/secflow-validator`

## Repository Structure

- `cmd/collector/main.go` - Collector service entry point
- `cmd/validator/main.go` - Validator service entry point
- `internal/collector/` - Collector service logic
- `internal/validator/` - Validator service logic
- `internal/config/` - Shared configuration management
- `deployments/collector/Dockerfile` - Collector container build
- `deployments/validator/Dockerfile` - Validator container build
- `Makefile` - Development and deployment commands
- `go.mod/go.sum` - Go module definitions

## Message Flow

1. **Collector** scans GitHub organization repositories and publishes to `github.repositories`
2. **Validator** consumes from `github.repositories`, checks for `appsec-config.yml`, and routes to:
   - `repos.valid` - repositories with the configuration file
   - `repos.invalid` - repositories without the configuration file

## Message Format

Repository messages are in JSON format with this structure:
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