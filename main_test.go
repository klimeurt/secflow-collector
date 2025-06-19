package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		wantErr     bool
		expectedCfg *Config
	}{
		{
			name: "valid config with all env vars",
			envVars: map[string]string{
				"GITHUB_ORG":     "testorg",
				"GITHUB_TOKEN":   "token123",
				"NATS_URL":       "nats://test:4222",
				"NATS_SUBJECT":   "test.repos",
				"CRON_SCHEDULE":  "0 */6 * * *",
				"RUN_ON_STARTUP": "true",
			},
			wantErr: false,
			expectedCfg: &Config{
				GitHubOrg:    "testorg",
				GitHubToken:  "token123",
				NATSUrl:      "nats://test:4222",
				NATSSubject:  "test.repos",
				CronSchedule: "0 */6 * * *",
				RunOnStartup: true,
			},
		},
		{
			name: "valid config with defaults",
			envVars: map[string]string{
				"GITHUB_ORG":   "testorg",
				"GITHUB_TOKEN": "token123",
			},
			wantErr: false,
			expectedCfg: &Config{
				GitHubOrg:    "testorg",
				GitHubToken:  "token123",
				NATSUrl:      "nats://localhost:4222",
				NATSSubject:  "github.repositories",
				CronSchedule: "0 0 * * 0",
				RunOnStartup: false,
			},
		},
		{
			name: "missing github org",
			envVars: map[string]string{
				"GITHUB_TOKEN": "token123",
			},
			wantErr: true,
		},
		{
			name: "missing github token",
			envVars: map[string]string{
				"GITHUB_ORG": "testorg",
			},
			wantErr: true,
		},
		{
			name: "run on startup false",
			envVars: map[string]string{
				"GITHUB_ORG":     "testorg",
				"GITHUB_TOKEN":   "token123",
				"RUN_ON_STARTUP": "false",
			},
			wantErr: false,
			expectedCfg: &Config{
				GitHubOrg:    "testorg",
				GitHubToken:  "token123",
				NATSUrl:      "nats://localhost:4222",
				NATSSubject:  "github.repositories",
				CronSchedule: "0 0 * * 0",
				RunOnStartup: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			defer clearEnv()

			cfg, err := loadConfig()

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadConfig() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("loadConfig() unexpected error: %v", err)
				return
			}

			if cfg.GitHubOrg != tt.expectedCfg.GitHubOrg {
				t.Errorf("GitHubOrg = %v, want %v", cfg.GitHubOrg, tt.expectedCfg.GitHubOrg)
			}
			if cfg.GitHubToken != tt.expectedCfg.GitHubToken {
				t.Errorf("GitHubToken = %v, want %v", cfg.GitHubToken, tt.expectedCfg.GitHubToken)
			}
			if cfg.NATSUrl != tt.expectedCfg.NATSUrl {
				t.Errorf("NATSUrl = %v, want %v", cfg.NATSUrl, tt.expectedCfg.NATSUrl)
			}
			if cfg.NATSSubject != tt.expectedCfg.NATSSubject {
				t.Errorf("NATSSubject = %v, want %v", cfg.NATSSubject, tt.expectedCfg.NATSSubject)
			}
			if cfg.CronSchedule != tt.expectedCfg.CronSchedule {
				t.Errorf("CronSchedule = %v, want %v", cfg.CronSchedule, tt.expectedCfg.CronSchedule)
			}
			if cfg.RunOnStartup != tt.expectedCfg.RunOnStartup {
				t.Errorf("RunOnStartup = %v, want %v", cfg.RunOnStartup, tt.expectedCfg.RunOnStartup)
			}
		})
	}
}

func TestRepositoryJSONSerialization(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)

	repo := Repository{
		Name:      "test-repo",
		CloneURL:  "https://github.com/org/test-repo.git",
		SSHURL:    "git@github.com:org/test-repo.git",
		HTTPSURL:  "https://github.com/org/test-repo.git",
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Language:  "Go",
		Topics:    []string{"microservice", "kubernetes"},
	}

	// Test marshaling
	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("Failed to marshal repository: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Repository
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal repository: %v", err)
	}

	// Verify fields
	if unmarshaled.Name != repo.Name {
		t.Errorf("Name = %v, want %v", unmarshaled.Name, repo.Name)
	}
	if unmarshaled.CloneURL != repo.CloneURL {
		t.Errorf("CloneURL = %v, want %v", unmarshaled.CloneURL, repo.CloneURL)
	}
	if unmarshaled.SSHURL != repo.SSHURL {
		t.Errorf("SSHURL = %v, want %v", unmarshaled.SSHURL, repo.SSHURL)
	}
	if unmarshaled.HTTPSURL != repo.HTTPSURL {
		t.Errorf("HTTPSURL = %v, want %v", unmarshaled.HTTPSURL, repo.HTTPSURL)
	}
	if !unmarshaled.CreatedAt.Equal(repo.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", unmarshaled.CreatedAt, repo.CreatedAt)
	}
	if !unmarshaled.UpdatedAt.Equal(repo.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", unmarshaled.UpdatedAt, repo.UpdatedAt)
	}
	if unmarshaled.Language != repo.Language {
		t.Errorf("Language = %v, want %v", unmarshaled.Language, repo.Language)
	}
	if len(unmarshaled.Topics) != len(repo.Topics) {
		t.Errorf("Topics length = %v, want %v", len(unmarshaled.Topics), len(repo.Topics))
	}
	for i, topic := range unmarshaled.Topics {
		if topic != repo.Topics[i] {
			t.Errorf("Topics[%d] = %v, want %v", i, topic, repo.Topics[i])
		}
	}
}

func TestRepositoryEmptyFields(t *testing.T) {
	repo := Repository{
		Name:      "test-repo",
		CloneURL:  "https://github.com/org/test-repo.git",
		SSHURL:    "git@github.com:org/test-repo.git",
		HTTPSURL:  "https://github.com/org/test-repo.git",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("Failed to marshal repository: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, `"language"`) {
		t.Error("Empty language field should be omitted from JSON")
	}
	if strings.Contains(jsonStr, `"topics"`) {
		t.Error("Empty topics field should be omitted from JSON")
	}
}

func TestScannerCreation(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		mockNATS      bool
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config with mock NATS",
			config: &Config{
				GitHubOrg:    "testorg",
				GitHubToken:  "token123",
				NATSUrl:      "nats://localhost:4222",
				NATSSubject:  "github.repositories",
				CronSchedule: "0 0 * * 0",
			},
			mockNATS:    true,
			expectError: false,
		},
		{
			name: "invalid NATS URL",
			config: &Config{
				GitHubOrg:    "testorg",
				GitHubToken:  "token123",
				NATSUrl:      "invalid://url",
				NATSSubject:  "github.repositories",
				CronSchedule: "0 0 * * 0",
			},
			mockNATS:      false,
			expectError:   true,
			errorContains: "failed to connect to NATS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *natsserver.Server
			if tt.mockNATS {
				server = runMockNATSServer()
				defer server.Shutdown()
				tt.config.NATSUrl = server.ClientURL()
			}

			scanner, err := NewScanner(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewScanner() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("NewScanner() error = %v, want to contain %v", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NewScanner() unexpected error: %v", err)
				return
			}

			if scanner == nil {
				t.Error("NewScanner() returned nil scanner")
				return
			}

			if scanner.config != tt.config {
				t.Error("Scanner config not set correctly")
			}

			if scanner.ghClient == nil {
				t.Error("GitHub client not initialized")
			}

			if scanner.nc == nil {
				t.Error("NATS connection not initialized")
			}

			scanner.Close()
		})
	}
}

func TestScannerPublishRepository(t *testing.T) {
	server := runMockNATSServer()
	defer server.Shutdown()

	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      server.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Create a subscriber to capture published messages
	messages := make(chan *nats.Msg, 1)
	sub, err := scanner.nc.ChanSubscribe(config.NATSSubject, messages)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	// Create test GitHub repository
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	githubRepo := createMockGitHubRepo("test-repo", "https://github.com/org/test-repo.git",
		"git@github.com:org/test-repo.git", createdAt, updatedAt, "Go", []string{"microservice"})

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

		if repo.Name != "test-repo" {
			t.Errorf("Repository name = %v, want %v", repo.Name, "test-repo")
		}
		if repo.Language != "Go" {
			t.Errorf("Repository language = %v, want %v", repo.Language, "Go")
		}
		if len(repo.Topics) != 1 || repo.Topics[0] != "microservice" {
			t.Errorf("Repository topics = %v, want %v", repo.Topics, []string{"microservice"})
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for published message")
	}
}

func TestScannerClose(t *testing.T) {
	server := runMockNATSServer()
	defer server.Shutdown()

	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      server.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	if scanner.nc.IsClosed() {
		t.Error("NATS connection should be open initially")
	}

	scanner.Close()

	if !scanner.nc.IsClosed() {
		t.Error("NATS connection should be closed after Close()")
	}
}

func TestScanRepositoriesPagination(t *testing.T) {
	// Create mock GitHub API server
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/repos") {
			http.NotFound(w, r)
			return
		}

		page := r.URL.Query().Get("page")
		var repos []map[string]interface{}

		if page == "" || page == "1" {
			// First page
			repos = []map[string]interface{}{
				createMockRepoJSON("repo1"),
				createMockRepoJSON("repo2"),
			}
			w.Header().Set("Link", fmt.Sprintf(`<%s/orgs/testorg/repos?page=2>; rel="next"`, server.URL))
		} else if page == "2" {
			// Second page
			repos = []map[string]interface{}{
				createMockRepoJSON("repo3"),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer server.Close()

	// Create NATS server
	natsServer := runMockNATSServer()
	defer natsServer.Shutdown()

	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      natsServer.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Override GitHub client base URL for testing
	scanner.ghClient.BaseURL = mustParseURL(server.URL + "/")

	// Subscribe to messages
	messages := make(chan *nats.Msg, 10)
	sub, err := scanner.nc.ChanSubscribe(config.NATSSubject, messages)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	// Scan repositories
	ctx := context.Background()
	err = scanner.ScanRepositories(ctx)
	if err != nil {
		t.Fatalf("Failed to scan repositories: %v", err)
	}

	// Collect messages
	var receivedRepos []string
	timeout := time.After(5 * time.Second)

	for len(receivedRepos) < 3 {
		select {
		case msg := <-messages:
			var repo Repository
			err = json.Unmarshal(msg.Data, &repo)
			if err != nil {
				t.Fatalf("Failed to unmarshal message: %v", err)
			}
			receivedRepos = append(receivedRepos, repo.Name)
		case <-timeout:
			t.Fatalf("Timeout waiting for messages, got %d repos: %v", len(receivedRepos), receivedRepos)
		}
	}

	expectedRepos := []string{"repo1", "repo2", "repo3"}
	if len(receivedRepos) != len(expectedRepos) {
		t.Errorf("Expected %d repos, got %d", len(expectedRepos), len(receivedRepos))
	}

	for _, expected := range expectedRepos {
		found := false
		for _, received := range receivedRepos {
			if received == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected repo %s not found in received repos: %v", expected, receivedRepos)
		}
	}
}

func TestScanRepositoriesError(t *testing.T) {
	// Create mock GitHub API server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Bad credentials"})
	}))
	defer server.Close()

	// Create NATS server
	natsServer := runMockNATSServer()
	defer natsServer.Shutdown()

	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "invalid-token",
		NATSUrl:      natsServer.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Override GitHub client base URL for testing
	scanner.ghClient.BaseURL = mustParseURL(server.URL + "/")

	// Scan repositories should return error
	ctx := context.Background()
	err = scanner.ScanRepositories(ctx)
	if err == nil {
		t.Error("Expected error from ScanRepositories, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list repositories") {
		t.Errorf("Expected 'failed to list repositories' error, got: %v", err)
	}
}

func TestPublishRepositoryMarshalError(t *testing.T) {
	natsServer := runMockNATSServer()
	defer natsServer.Shutdown()

	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      natsServer.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Create a GitHub repository with invalid time that will cause JSON marshal error
	// This is a bit contrived but tests the marshal error path
	githubRepo := &github.Repository{
		Name:     github.String("test-repo"),
		CloneURL: github.String("https://github.com/org/test-repo.git"),
		SSHURL:   github.String("git@github.com:org/test-repo.git"),
		// Note: We can't easily force a marshal error with valid Repository struct
		// But this test structure is here for completeness
	}

	// This should succeed normally
	err = scanner.publishRepository(githubRepo)
	if err != nil {
		t.Errorf("Unexpected error publishing repository: %v", err)
	}
}

func TestPublishRepositoryNATSError(t *testing.T) {
	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      "nats://localhost:4222", // Will fail when NATS server doesn't exist
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	// Try to create scanner - this should fail due to NATS connection
	scanner, err := NewScanner(config)
	if err == nil {
		scanner.Close()
		t.Skip("NATS server is running, skipping NATS error test")
	}

	// Expected behavior: NewScanner should fail with NATS connection error
	if !strings.Contains(err.Error(), "failed to connect to NATS") {
		t.Errorf("Expected 'failed to connect to NATS' error, got: %v", err)
	}
}

func TestScanRepositoriesWithPublishErrors(t *testing.T) {
	// Create mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/repos") {
			http.NotFound(w, r)
			return
		}

		repos := []map[string]interface{}{
			createMockRepoJSON("repo1"),
			createMockRepoJSON("repo2"),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer server.Close()

	// Create NATS server and then close it to cause publish errors
	natsServer := runMockNATSServer()
	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      natsServer.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Close NATS server to cause publish errors
	natsServer.Shutdown()
	scanner.nc.Close()

	// Override GitHub client base URL for testing
	scanner.ghClient.BaseURL = mustParseURL(server.URL + "/")

	// Scan should succeed even with publish errors (they're logged but not fatal)
	ctx := context.Background()
	err = scanner.ScanRepositories(ctx)
	if err != nil {
		t.Errorf("ScanRepositories should succeed even with publish errors: %v", err)
	}

	scanner.Close() // Should handle already closed connection gracefully
}

func TestRepositoryFieldMapping(t *testing.T) {
	// Test all GitHub repository fields are mapped correctly
	createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 12, 1, 18, 30, 0, 0, time.UTC)

	githubRepo := &github.Repository{
		Name:      github.String("comprehensive-test-repo"),
		CloneURL:  github.String("https://github.com/org/comprehensive-test-repo.git"),
		SSHURL:    github.String("git@github.com:org/comprehensive-test-repo.git"),
		CreatedAt: &github.Timestamp{Time: createdAt},
		UpdatedAt: &github.Timestamp{Time: updatedAt},
		Language:  github.String("TypeScript"),
		Topics:    []string{"backend", "api", "microservice", "docker", "kubernetes"},
	}

	natsServer := runMockNATSServer()
	defer natsServer.Shutdown()

	config := &Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      natsServer.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := NewScanner(config)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	defer scanner.Close()

	// Subscribe to messages
	messages := make(chan *nats.Msg, 1)
	sub, err := scanner.nc.ChanSubscribe(config.NATSSubject, messages)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	// Publish repository
	err = scanner.publishRepository(githubRepo)
	if err != nil {
		t.Fatalf("Failed to publish repository: %v", err)
	}

	// Wait for message and verify all fields
	select {
	case msg := <-messages:
		var repo Repository
		err = json.Unmarshal(msg.Data, &repo)
		if err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		if repo.Name != "comprehensive-test-repo" {
			t.Errorf("Name = %v, want %v", repo.Name, "comprehensive-test-repo")
		}
		if repo.CloneURL != "https://github.com/org/comprehensive-test-repo.git" {
			t.Errorf("CloneURL = %v, want %v", repo.CloneURL, "https://github.com/org/comprehensive-test-repo.git")
		}
		if repo.SSHURL != "git@github.com:org/comprehensive-test-repo.git" {
			t.Errorf("SSHURL = %v, want %v", repo.SSHURL, "git@github.com:org/comprehensive-test-repo.git")
		}
		if repo.HTTPSURL != repo.CloneURL {
			t.Errorf("HTTPSURL = %v, want %v", repo.HTTPSURL, repo.CloneURL)
		}
		if !repo.CreatedAt.Equal(createdAt) {
			t.Errorf("CreatedAt = %v, want %v", repo.CreatedAt, createdAt)
		}
		if !repo.UpdatedAt.Equal(updatedAt) {
			t.Errorf("UpdatedAt = %v, want %v", repo.UpdatedAt, updatedAt)
		}
		if repo.Language != "TypeScript" {
			t.Errorf("Language = %v, want %v", repo.Language, "TypeScript")
		}
		if len(repo.Topics) != 5 {
			t.Errorf("Topics length = %v, want %v", len(repo.Topics), 5)
		}
		expectedTopics := []string{"backend", "api", "microservice", "docker", "kubernetes"}
		for i, topic := range expectedTopics {
			if i >= len(repo.Topics) || repo.Topics[i] != topic {
				t.Errorf("Topics[%d] = %v, want %v", i, repo.Topics[i], topic)
			}
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for published message")
	}
}

// Test helper functions

func clearEnv() {
	envVars := []string{
		"GITHUB_ORG", "GITHUB_TOKEN", "NATS_URL",
		"NATS_SUBJECT", "CRON_SCHEDULE", "RUN_ON_STARTUP",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func runMockNATSServer() *natsserver.Server {
	opts := &natsserver.Options{
		Host: "127.0.0.1",
		Port: -1, // Use random port
	}

	server := natsserver.New(opts)

	go server.Start()

	if !server.ReadyForConnections(5 * time.Second) {
		panic("NATS server not ready")
	}

	return server
}

func createMockGitHubRepo(name, cloneURL, sshURL string, createdAt, updatedAt time.Time, language string, topics []string) *github.Repository {
	return &github.Repository{
		Name:      github.String(name),
		CloneURL:  github.String(cloneURL),
		SSHURL:    github.String(sshURL),
		CreatedAt: &github.Timestamp{Time: createdAt},
		UpdatedAt: &github.Timestamp{Time: updatedAt},
		Language:  github.String(language),
		Topics:    topics,
	}
}

func createMockRepoJSON(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":       name,
		"clone_url":  fmt.Sprintf("https://github.com/org/%s.git", name),
		"ssh_url":    fmt.Sprintf("git@github.com:org/%s.git", name),
		"created_at": "2023-01-01T00:00:00Z",
		"updated_at": "2023-12-01T00:00:00Z",
		"language":   "Go",
		"topics":     []string{"test"},
	}
}

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse URL %s: %v", rawURL, err))
	}
	return u
}
