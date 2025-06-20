package collector

import "time"

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