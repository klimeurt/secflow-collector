package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/klimeurt/secflow-collector/internal/config"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func TestScannerCreation(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		mockNATS      bool
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config with mock NATS",
			config: &config.Config{
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
			config: &config.Config{
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

			scanner, err := New(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("New() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("New() error = %v, want to contain %v", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}

			if scanner == nil {
				t.Error("New() returned nil scanner")
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

	config := &config.Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      server.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := New(config)
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

	config := &config.Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      server.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := New(config)
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

	config := &config.Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "token123",
		NATSUrl:      natsServer.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := New(config)
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

	config := &config.Config{
		GitHubOrg:    "testorg",
		GitHubToken:  "invalid-token",
		NATSUrl:      natsServer.ClientURL(),
		NATSSubject:  "github.repositories",
		CronSchedule: "0 0 * * 0",
	}

	scanner, err := New(config)
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

// Test helper functions

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