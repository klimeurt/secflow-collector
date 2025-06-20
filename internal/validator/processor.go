package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/klimeurt/secflow-collector/internal/collector"
	"github.com/klimeurt/secflow-collector/internal/config"
	"github.com/nats-io/nats.go"
)

// Processor handles message processing and routing
type Processor struct {
	config  *config.Config
	checker *Checker
	nc      *nats.Conn
}

// NewProcessor creates a new Processor instance
func NewProcessor(cfg *config.Config, checker *Checker, nc *nats.Conn) *Processor {
	return &Processor{
		config:  cfg,
		checker: checker,
		nc:      nc,
	}
}

// ProcessMessage processes a repository message and routes it to appropriate queue
func (p *Processor) ProcessMessage(ctx context.Context, msg *nats.Msg) error {
	// Parse the repository message
	var repo collector.Repository
	if err := json.Unmarshal(msg.Data, &repo); err != nil {
		return fmt.Errorf("failed to unmarshal repository message: %w", err)
	}

	log.Printf("Processing repository: %s", repo.Name)

	// Extract owner from clone URL
	owner, err := p.extractOwnerFromURL(repo.CloneURL)
	if err != nil {
		return fmt.Errorf("failed to extract owner from URL %s: %w", repo.CloneURL, err)
	}

	// Check if repository has appsec-config.yml
	hasConfig, err := p.checker.HasAppSecConfig(ctx, owner, repo.Name)
	if err != nil {
		log.Printf("Error checking appsec-config.yml for %s/%s: %v", owner, repo.Name, err)
		// Route to invalid queue on error
		hasConfig = false
	}

	// Route message to appropriate queue
	var targetSubject string
	if hasConfig {
		targetSubject = p.config.ValidReposSubject
		log.Printf("Repository %s has appsec-config.yml - routing to %s", repo.Name, targetSubject)
	} else {
		targetSubject = p.config.InvalidReposSubject
		log.Printf("Repository %s missing appsec-config.yml - routing to %s", repo.Name, targetSubject)
	}

	// Publish to target queue
	if err := p.nc.Publish(targetSubject, msg.Data); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", targetSubject, err)
	}

	return nil
}

// extractOwnerFromURL extracts the owner/organization from a GitHub clone URL
func (p *Processor) extractOwnerFromURL(cloneURL string) (string, error) {
	// Example URLs:
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	
	if strings.HasPrefix(cloneURL, "https://github.com/") {
		// Remove prefix and suffix
		path := strings.TrimPrefix(cloneURL, "https://github.com/")
		path = strings.TrimSuffix(path, ".git")
		
		// Split by / and get the first part (owner)
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			return parts[0], nil
		}
	} else if strings.HasPrefix(cloneURL, "git@github.com:") {
		// Remove prefix and suffix
		path := strings.TrimPrefix(cloneURL, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		
		// Split by / and get the first part (owner)
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			return parts[0], nil
		}
	}
	
	return "", fmt.Errorf("unable to parse owner from URL: %s", cloneURL)
}