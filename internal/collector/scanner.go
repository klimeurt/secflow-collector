package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/go-github/v57/github"
	"github.com/klimeurt/secflow-collector/internal/config"
	"github.com/nats-io/nats.go"
	"golang.org/x/oauth2"
)

// Scanner handles the GitHub scanning operations
type Scanner struct {
	config   *config.Config
	ghClient *github.Client
	nc       *nats.Conn
}

// New creates a new Scanner instance
func New(cfg *config.Config) (*Scanner, error) {
	// Create GitHub client with OAuth2 token
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.GitHubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	// Connect to NATS
	nc, err := nats.Connect(cfg.NATSUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Scanner{
		config:   cfg,
		ghClient: ghClient,
		nc:       nc,
	}, nil
}

// ScanRepositories fetches all repositories from the GitHub organization
func (s *Scanner) ScanRepositories(ctx context.Context) error {
	log.Printf("Starting repository scan for organization: %s", s.config.GitHubOrg)

	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := s.ghClient.Repositories.ListByOrg(ctx, s.config.GitHubOrg, opt)
		if err != nil {
			return fmt.Errorf("failed to list repositories: %w", err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	log.Printf("Found %d repositories", len(allRepos))

	// Process and publish each repository
	for _, repo := range allRepos {
		if err := s.publishRepository(repo); err != nil {
			log.Printf("Failed to publish repository %s: %v", repo.GetName(), err)
			// Continue processing other repositories
		}
	}

	log.Printf("Successfully processed %d repositories", len(allRepos))
	return nil
}

// publishRepository publishes a repository to the NATS queue
func (s *Scanner) publishRepository(repo *github.Repository) error {
	// Convert GitHub repository to our Repository struct
	r := Repository{
		Name:      repo.GetName(),
		CloneURL:  repo.GetCloneURL(),
		SSHURL:    repo.GetSSHURL(),
		HTTPSURL:  repo.GetCloneURL(),
		CreatedAt: repo.GetCreatedAt().Time,
		UpdatedAt: repo.GetUpdatedAt().Time,
		Language:  repo.GetLanguage(),
		Topics:    repo.Topics,
	}

	// Serialize to JSON
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal repository: %w", err)
	}

	// Publish to NATS
	if err := s.nc.Publish(s.config.NATSSubject, data); err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}

	log.Printf("Published repository: %s", r.Name)
	return nil
}

// Close cleanly shuts down the scanner
func (s *Scanner) Close() {
	if s.nc != nil {
		s.nc.Close()
	}
}