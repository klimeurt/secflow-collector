# GitHub Repository Scanner

A Go microservice that periodically scans a GitHub organization's repositories and publishes them to a NATS queue for further processing.

## Features

- **Scheduled Scanning**: Uses cron-style scheduling (customizable, defaults to weekly)
- **GitHub Integration**: Fetches all repositories from a specified organization
- **NATS Publishing**: Publishes repository information to a NATS queue
- **Kubernetes Ready**: Deployable via Helm with full Kubernetes support
- **Secure**: Runs as non-root user, supports security contexts
- **Configurable**: Environment variable based configuration

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│                 │     │                 │     │                 │
│  GitHub API     │◄────│  Scanner Pod    │────►│   NATS Queue    │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                              │
                              ▼
                        ┌─────────────┐
                        │   Cron      │
                        │  Scheduler  │
                        └─────────────┘
```

## Repository Message Format

Each repository is published to NATS as a JSON message:

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

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `GITHUB_ORG` | GitHub organization name | - | Yes |
| `GITHUB_TOKEN` | GitHub personal access token | - | Yes |
| `NATS_URL` | NATS server URL | `nats://localhost:4222` | No |
| `NATS_SUBJECT` | NATS subject for publishing | `github.repositories` | No |
| `CRON_SCHEDULE` | Cron schedule expression | `0 0 * * 0` (weekly) | No |
| `RUN_ON_STARTUP` | Run scan immediately on startup | `false` | No |

### Cron Schedule Examples

- `0 0 * * 0` - Every Sunday at midnight (default)
- `0 2 * * *` - Daily at 2 AM
- `*/30 * * * *` - Every 30 minutes
- `0 9-17 * * 1-5` - Every hour from 9 AM to 5 PM on weekdays

## Local Development

### Prerequisites

- Go 1.21+
- Docker
- NATS server (for testing)

### Running Locally

1. Start NATS server:
```bash
docker run -d --name nats -p 4222:4222 nats:latest
```

2. Set environment variables:
```bash
export GITHUB_ORG="your-org"
export GITHUB_TOKEN="your-github-token"
export NATS_URL="nats://localhost:4222"
export RUN_ON_STARTUP="true"
```

3. Run the application:
```bash
go run main.go
```

### Building

```bash
# Build binary
go build -o secflow-collectortor main.go

# Build Docker image
docker build -t secflow-collector:latest .
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster
- Helm 3.x
- NATS deployed in the cluster

### Create GitHub Token Secret

```bash
kubectl create secret generic github-token \
  --from-literal=token=your-github-token
```

### Install with Helm

1. Create values file (`my-values.yaml`):
```yaml
github:
  organization: "your-org-name"
  tokenSecretName: "github-token"

nats:
  url: "nats://nats:4222"

cron:
  schedule: "0 0 * * 0"  # Weekly on Sunday
  runOnStartup: true     # Run immediately after deployment
```

2. Install the chart:
```bash
helm install secflow-collectortor ./helm/secflow-collector -f my-values.yaml
```

3. Upgrade deployment:
```bash
helm upgrade secflow-collector ./helm/secflow-collector -f my-values.yaml
```

4. Uninstall:
```bash
helm uninstall secflow-collector
```

### Helm Values Reference

See `helm/secflow-collector/values.yaml` for all available configuration options.

## Monitoring

The service logs all operations to stdout. You can view logs using:

```bash
kubectl logs -f deployment/secflow-collector
```

### Log Examples

```
2023/12/01 10:00:00 Cron scheduler started with schedule: 0 0 * * 0
2023/12/01 10:00:00 Running initial scan on startup...
2023/12/01 10:00:01 Starting repository scan for organization: example-org
2023/12/01 10:00:02 Found 25 repositories
2023/12/01 10:00:02 Published repository: repo-1
2023/12/01 10:00:02 Published repository: repo-2
...
2023/12/01 10:00:05 Successfully processed 25 repositories
```

## Security Considerations

1. **GitHub Token**: Store as Kubernetes secret, never in code
2. **Non-root User**: Container runs as UID 1000
3. **Read-only Filesystem**: Enabled by default in Helm chart
4. **Network Policies**: Consider implementing to restrict traffic
5. **RBAC**: ServiceAccount with minimal permissions

## Testing

### Unit Tests

```bash
go test ./... -v
```

### Integration Tests

```bash
# Requires running NATS server
go test ./... -tags=integration -v
```

## Directory Structure

```
.
├── main.go                 # Main application code
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── Dockerfile              # Multi-stage Docker build
├── README.md               # This file
└── helm/
    └── secflow-collector/
        ├── Chart.yaml      # Helm chart metadata
        ├── values.yaml     # Default values
        └── templates/
            ├── _helpers.tpl
            ├── deployment.yaml
            ├── serviceaccount.yaml
            └── NOTES.txt
```

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   - Verify GitHub token has `repo` scope
   - Check token hasn't expired

2. **NATS Connection Failed**
   - Verify NATS is running and accessible
   - Check network policies

3. **No Repositories Found**
   - Verify organization name is correct
   - Check token has access to the organization

4. **Cron Not Triggering**
   - Verify cron schedule syntax
   - Check logs for scheduler startup

### Debug Mode

Enable debug logging by setting:
```bash
export LOG_LEVEL=debug
```

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.