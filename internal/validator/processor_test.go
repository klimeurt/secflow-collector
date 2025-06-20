package validator

import (
	"testing"
)

func TestExtractOwnerFromURL(t *testing.T) {
	processor := &Processor{}
	
	tests := []struct {
		name     string
		url      string
		expected string
		hasError bool
	}{
		{
			name:     "HTTPS URL",
			url:      "https://github.com/klimeurt/secflow-collector.git",
			expected: "klimeurt",
			hasError: false,
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:klimeurt/secflow-collector.git",
			expected: "klimeurt",
			hasError: false,
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/klimeurt/secflow-collector",
			expected: "klimeurt",
			hasError: false,
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:klimeurt/secflow-collector",
			expected: "klimeurt",
			hasError: false,
		},
		{
			name:     "Invalid URL",
			url:      "invalid-url",
			expected: "",
			hasError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, err := processor.extractOwnerFromURL(tt.url)
			
			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for URL %s, but got none", tt.url)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for URL %s: %v", tt.url, err)
				}
				if owner != tt.expected {
					t.Errorf("Expected owner %s for URL %s, but got %s", tt.expected, tt.url, owner)
				}
			}
		})
	}
}