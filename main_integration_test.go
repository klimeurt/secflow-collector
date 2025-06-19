//go:build integration
// +build integration

package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/nats-io/nats.go"
)

func TestIntegrationGitHubAPI(t *testing.T) {
	// Skip if required environment variables are not set
	githubOrg := os.Getenv("GITHUB_ORG")
	githubToken := os.Getenv("GITHUB_TOKEN")

	if githubOrg == "" || githubToken == "" {
		t.Skip("Skipping integration test: GITHUB_ORG and GITHUB_TOKEN environment variables required")
	}

	config := &Config{
		GitHubOrg:    githubOrg,
		GitHubToken:  githubToken,
		NATSUrl:      "nats://localhost:4222",
		NATSSubject:  "github.repositories.test",
		CronSchedule: "0 0 * * 0",
	}

	// Test NATS connection
	nc, err := nats.Connect(config.NATSUrl)
	if err != nil {
		t.Skipf("Skipping integration test: NATS server not available at %s: %v", config.NATSUrl, err)
	}
	nc.Close()

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Test scanning repositories
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = scanner.ScanRepositories(ctx)
	if err != nil {
		t.Fatalf("Failed to scan repositories: %v", err)
	}

	t.Logf("Successfully scanned repositories for organization: %s", githubOrg)
}

func TestIntegrationNATSPublishing(t *testing.T) {
	// Test with local NATS server
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		t.Skipf("Skipping integration test: NATS server not available: %v", err)
	}
	defer nc.Close()

	subject := "github.repositories.integration.test"

	// Subscribe to messages
	messages := make(chan *nats.Msg, 10)
	sub, err := nc.ChanSubscribe(subject, messages)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "dummy-token",
		NATSUrl:      "nats://localhost:4222",
		NATSSubject:  subject,
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Create test repository
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	githubRepo := createMockGitHubRepo("integration-test-repo",
		"https://github.com/org/integration-test-repo.git",
		"git@github.com:org/integration-test-repo.git",
		createdAt, updatedAt, "Go", []string{"integration", "test"})

	// Publish repository
	err = scanner.publishRepository(githubRepo)
	if err != nil {
		t.Fatalf("Failed to publish repository: %v", err)
	}

	// Wait for message
	select {
	case msg := <-messages:
		var repo Repository
		err = json.Unmarshal(msg.Data, &repo)
		if err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		if repo.Name != "integration-test-repo" {
			t.Errorf("Repository name = %v, want %v", repo.Name, "integration-test-repo")
		}
		if repo.Language != "Go" {
			t.Errorf("Repository language = %v, want %v", repo.Language, "Go")
		}
		if len(repo.Topics) != 2 {
			t.Errorf("Repository topics length = %v, want %v", len(repo.Topics), 2)
		}

		t.Logf("Successfully published and received repository: %s", repo.Name)

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for published message")
	}
}

func TestIntegrationEndToEnd(t *testing.T) {
	// Skip if required environment variables are not set
	githubOrg := os.Getenv("GITHUB_ORG")
	githubToken := os.Getenv("GITHUB_TOKEN")

	if githubOrg == "" || githubToken == "" {
		t.Skip("Skipping integration test: GITHUB_ORG and GITHUB_TOKEN environment variables required")
	}

	// Test NATS connection
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		t.Skipf("Skipping integration test: NATS server not available: %v", err)
	}
	defer nc.Close()

	subject := "github.repositories.e2e.test"

	// Subscribe to messages
	messages := make(chan *nats.Msg, 100)
	sub, err := nc.ChanSubscribe(subject, messages)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	config := &Config{
		GitHubOrg:    githubOrg,
		GitHubToken:  githubToken,
		NATSUrl:      "nats://localhost:4222",
		NATSSubject:  subject,
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Scan repositories
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = scanner.ScanRepositories(ctx)
	if err != nil {
		t.Fatalf("Failed to scan repositories: %v", err)
	}

	// Collect messages for a short time to verify publishing works
	var receivedRepos []Repository
	timeout := time.After(10 * time.Second)

	for {
		select {
		case msg := <-messages:
			var repo Repository
			err = json.Unmarshal(msg.Data, &repo)
			if err != nil {
				t.Errorf("Failed to unmarshal message: %v", err)
				continue
			}
			receivedRepos = append(receivedRepos, repo)

			// Stop after receiving some repositories
			if len(receivedRepos) >= 5 {
				goto done
			}

		case <-timeout:
			goto done
		}
	}

done:
	if len(receivedRepos) == 0 {
		t.Fatal("No repositories received from end-to-end test")
	}

	t.Logf("End-to-end test completed successfully. Received %d repositories", len(receivedRepos))

	// Verify repository structure
	for i, repo := range receivedRepos {
		if repo.Name == "" {
			t.Errorf("Repository %d has empty name", i)
		}
		if repo.CloneURL == "" {
			t.Errorf("Repository %d has empty clone URL", i)
		}
		if repo.SSHURL == "" {
			t.Errorf("Repository %d has empty SSH URL", i)
		}
		if repo.HTTPSURL == "" {
			t.Errorf("Repository %d has empty HTTPS URL", i)
		}
		if repo.CreatedAt.IsZero() {
			t.Errorf("Repository %d has zero created_at time", i)
		}
		if repo.UpdatedAt.IsZero() {
			t.Errorf("Repository %d has zero updated_at time", i)
		}

		t.Logf("Repository %d: %s (Language: %s, Topics: %v)", i, repo.Name, repo.Language, repo.Topics)
	}
}

func TestIntegrationGitHubAPIRateLimit(t *testing.T) {
	// Skip if required environment variables are not set
	githubOrg := os.Getenv("GITHUB_ORG")
	githubToken := os.Getenv("GITHUB_TOKEN")

	if githubOrg == "" || githubToken == "" {
		t.Skip("Skipping integration test: GITHUB_ORG and GITHUB_TOKEN environment variables required")
	}

	config := &Config{
		GitHubOrg:    githubOrg,
		GitHubToken:  githubToken,
		NATSUrl:      "nats://localhost:4222",
		NATSSubject:  "github.repositories.ratelimit.test",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Check rate limit
	ctx := context.Background()
	rateLimit, _, err := scanner.ghClient.RateLimits(ctx)
	if err != nil {
		t.Fatalf("Failed to get rate limits: %v", err)
	}

	t.Logf("GitHub API Rate Limit - Core: %d/%d (resets at %v)",
		rateLimit.Core.Remaining, rateLimit.Core.Limit, rateLimit.Core.Reset.Time)

	if rateLimit.Core.Remaining < 10 {
		t.Skip("Skipping test: GitHub API rate limit too low")
	}

	// Perform a small scan to test rate limiting behavior
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	err = scanner.ScanRepositories(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to scan repositories: %v", err)
	}

	t.Logf("Repository scan completed in %v", duration)
}

func TestIntegrationNATSConnectionFailure(t *testing.T) {
	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "dummy-token",
		NATSUrl:      "nats://invalid-host:4222",
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	_, err := NewScanner(config)
	if err == nil {
		t.Error("Expected error when connecting to invalid NATS server")
	}

	if err != nil {
		t.Logf("Got expected error: %v", err)
	}
}

func TestIntegrationConfigFromEnvironment(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{"GITHUB_ORG", "GITHUB_TOKEN", "NATS_URL", "NATS_SUBJECT", "CRON_SCHEDULE", "RUN_ON_STARTUP"}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
	}

	defer func() {
		// Restore original environment
		for _, env := range envVars {
			if val, exists := originalEnv[env]; exists && val != "" {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	// Set test environment
	os.Setenv("GITHUB_ORG", "integration-test-org")
	os.Setenv("GITHUB_TOKEN", "integration-test-token")
	os.Setenv("NATS_URL", "nats://integration-test:4222")
	os.Setenv("NATS_SUBJECT", "integration.test.repos")
	os.Setenv("CRON_SCHEDULE", "0 */2 * * *")
	os.Setenv("RUN_ON_STARTUP", "true")

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.GitHubOrg != "integration-test-org" {
		t.Errorf("GitHubOrg = %v, want %v", config.GitHubOrg, "integration-test-org")
	}
	if config.GitHubToken != "integration-test-token" {
		t.Errorf("GitHubToken = %v, want %v", config.GitHubToken, "integration-test-token")
	}
	if config.NATSUrl != "nats://integration-test:4222" {
		t.Errorf("NATSUrl = %v, want %v", config.NATSUrl, "nats://integration-test:4222")
	}
	if config.NATSSubject != "integration.test.repos" {
		t.Errorf("NATSSubject = %v, want %v", config.NATSSubject, "integration.test.repos")
	}
	if config.CronSchedule != "0 */2 * * *" {
		t.Errorf("CronSchedule = %v, want %v", config.CronSchedule, "0 */2 * * *")
	}
	if !config.RunOnStartup {
		t.Errorf("RunOnStartup = %v, want %v", config.RunOnStartup, true)
	}

	t.Logf("Integration config test passed with all environment variables")
}
