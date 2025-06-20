package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
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

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
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

func clearEnv() {
	envVars := []string{
		"GITHUB_ORG", "GITHUB_TOKEN", "NATS_URL",
		"NATS_SUBJECT", "CRON_SCHEDULE", "RUN_ON_STARTUP",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}