package collector

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

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