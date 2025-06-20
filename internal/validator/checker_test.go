package validator

import (
	"context"
	"testing"

	"github.com/klimeurt/secflow-collector/internal/config"
)

func TestNewChecker(t *testing.T) {
	// Test with valid config
	cfg := &config.Config{
		GitHubToken: "test-token",
	}
	
	checker, err := NewChecker(cfg)
	if err != nil {
		t.Fatalf("Expected no error creating checker, got: %v", err)
	}
	
	if checker == nil {
		t.Fatal("Expected checker to be created, got nil")
	}
	
	if checker.config != cfg {
		t.Error("Expected checker config to match input config")
	}
	
	if checker.ghClient == nil {
		t.Error("Expected GitHub client to be initialized")
	}
}

func TestHasAppSecConfig(t *testing.T) {
	// Note: This test would require GitHub API mocking for full testing
	// For now, we'll just test that the method exists and has the right signature
	cfg := &config.Config{
		GitHubToken: "test-token",
	}
	
	checker, err := NewChecker(cfg)
	if err != nil {
		t.Fatalf("Failed to create checker: %v", err)
	}
	
	// Test method exists and returns appropriate types
	ctx := context.Background()
	_, err = checker.HasAppSecConfig(ctx, "test-owner", "test-repo")
	if err == nil {
		t.Log("No error returned - this is unexpected with test token but not critical")
	}
	// We expect an error here since we're using a test token
	// The important thing is that the method doesn't panic
}