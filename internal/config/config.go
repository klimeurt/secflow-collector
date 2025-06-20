package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	GitHubOrg           string
	GitHubToken         string
	NATSUrl             string
	NATSSubject         string
	CronSchedule        string
	RunOnStartup        bool
	// Validator specific configuration
	ValidReposSubject     string
	InvalidReposSubject   string
	SourceSubject         string
	ProcessStartupMessages bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		GitHubOrg:           os.Getenv("GITHUB_ORG"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		NATSUrl:             os.Getenv("NATS_URL"),
		NATSSubject:         os.Getenv("NATS_SUBJECT"),
		CronSchedule:        os.Getenv("CRON_SCHEDULE"),
		ValidReposSubject:   os.Getenv("VALID_REPOS_SUBJECT"),
		InvalidReposSubject: os.Getenv("INVALID_REPOS_SUBJECT"),
		SourceSubject:       os.Getenv("SOURCE_SUBJECT"),
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
	if cfg.ValidReposSubject == "" {
		cfg.ValidReposSubject = "repos.valid"
	}
	if cfg.InvalidReposSubject == "" {
		cfg.InvalidReposSubject = "repos.invalid"
	}
	if cfg.SourceSubject == "" {
		cfg.SourceSubject = "github.repositories"
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

	// Check if we should process startup messages (default: true)
	if os.Getenv("PROCESS_STARTUP_MESSAGES") == "false" {
		cfg.ProcessStartupMessages = false
	} else {
		cfg.ProcessStartupMessages = true
	}

	return cfg, nil
}