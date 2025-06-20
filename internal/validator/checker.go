package validator

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v57/github"
	"github.com/klimeurt/secflow-collector/internal/config"
	"golang.org/x/oauth2"
)

// Checker handles GitHub API operations for file validation
type Checker struct {
	config   *config.Config
	ghClient *github.Client
}

// NewChecker creates a new Checker instance
func NewChecker(cfg *config.Config) (*Checker, error) {
	// Create GitHub client with OAuth2 token
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.GitHubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	return &Checker{
		config:   cfg,
		ghClient: ghClient,
	}, nil
}

// HasAppSecConfig checks if the repository has an appsec-config.yml file in the root
func (c *Checker) HasAppSecConfig(ctx context.Context, owner, repo string) (bool, error) {
	// Try to get the file content to check if it exists
	_, _, resp, err := c.ghClient.Repositories.GetContents(
		ctx,
		owner,
		repo,
		"appsec-config.yml",
		&github.RepositoryContentGetOptions{},
	)

	if err != nil {
		// Check if it's a 404 error (file not found)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check for appsec-config.yml: %w", err)
	}

	return true, nil
}