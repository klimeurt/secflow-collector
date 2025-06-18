package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/nats-io/nats.go"
	"github.com/robfig/cron/v3"
	"golang.org/x/oauth2"
)

// Config holds the application configuration
type Config struct {
	GitHubOrg    string
	GitHubToken  string
	NATSUrl      string
	NATSSubject  string
	CronSchedule string
	RunOnStartup bool
}

// Repository represents a GitHub repository
type Repository struct {
	Name      string    `json:"name"`
	CloneURL  string    `json:"clone_url"`
	SSHURL    string    `json:"ssh_url"`
	HTTPSURL  string    `json:"https_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Language  string    `json:"language,omitempty"`
	Topics    []string  `json:"topics,omitempty"`
}

// Scanner handles the GitHub scanning operations
type Scanner struct {
	config   *Config
	ghClient *github.Client
	nc       *nats.Conn
}

// NewScanner creates a new Scanner instance
func NewScanner(cfg *Config) (*Scanner, error) {
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

// loadConfig loads configuration from environment variables
func loadConfig() (*Config, error) {
	cfg := &Config{
		GitHubOrg:    os.Getenv("GITHUB_ORG"),
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		NATSUrl:      os.Getenv("NATS_URL"),
		NATSSubject:  os.Getenv("NATS_SUBJECT"),
		CronSchedule: os.Getenv("CRON_SCHEDULE"),
	}

	// Set defaults
	if cfg.NATSUrl == "" {
		cfg.NATSUrl = "nats://localhost:4222"
	}
	if cfg.NATSSubject == "" {
		cfg.NATSSubject = "github.repositories"
	}
	if cfg.CronSchedule == "" {
		cfg.CronSchedule = "0 0 * * 0" // Weekly on Sunday at midnight
	}

	// Validate required fields
	if cfg.GitHubOrg == "" {
		return nil, fmt.Errorf("GITHUB_ORG environment variable is required")
	}
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	// Check if we should run on startup
	if os.Getenv("RUN_ON_STARTUP") == "true" {
		cfg.RunOnStartup = true
	}

	return cfg, nil
}

func main() {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create scanner
	scanner, err := NewScanner(cfg)
	if err != nil {
		log.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Create cron scheduler
	c := cron.New()

	// Add job
	_, err = c.AddFunc(cfg.CronSchedule, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := scanner.ScanRepositories(ctx); err != nil {
			log.Printf("Scan failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}

	// Start cron scheduler
	c.Start()
	log.Printf("Cron scheduler started with schedule: %s", cfg.CronSchedule)

	// Run immediately on startup if configured
	if cfg.RunOnStartup {
		log.Println("Running initial scan on startup...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := scanner.ScanRepositories(ctx); err != nil {
			log.Printf("Initial scan failed: %v", err)
		}
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	c.Stop()
}
