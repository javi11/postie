package config

import (
	"os"
	"path/filepath"
	"testing"
)

// validBaseConfig returns a minimal config that passes Validate(), so individual
// tests can mutate a single field and assert the validation outcome.
func validBaseConfig() ConfigData {
	enabled := true
	cfg := GetDefaultConfig()
	cfg.Servers = []ServerConfig{
		{
			Host:           "news.example.com",
			Port:           563,
			SSL:            true,
			MaxConnections: 10,
			Enabled:        &enabled,
			Role:           ServerRoleUpload,
		},
	}
	return cfg
}

func TestGetDefaultConfig_Phase0Fields(t *testing.T) {
	cfg := GetDefaultConfig()

	if cfg.Par2.MaxConcurrentJobs != 1 {
		t.Errorf("Par2.MaxConcurrentJobs default = %d, want 1", cfg.Par2.MaxConcurrentJobs)
	}
	if cfg.Posting.UploadBufferMemoryLimit != 0 {
		t.Errorf("Posting.UploadBufferMemoryLimit default = %d, want 0 (auto)", cfg.Posting.UploadBufferMemoryLimit)
	}
	if cfg.PostCheck.MaxConcurrentChecks != 0 {
		t.Errorf("PostCheck.MaxConcurrentChecks default = %d, want 0 (auto)", cfg.PostCheck.MaxConcurrentChecks)
	}
}

// TestLoad_AppliesPar2MaxConcurrentJobsDefault ensures a config file that omits
// par2.max_concurrent_jobs is normalised to 1 (so MemoryLimit is not multiplied
// across simultaneous queue jobs).
func TestLoad_AppliesPar2MaxConcurrentJobsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `servers:
  - host: news.example.com
    port: 563
    ssl: true
    max_connections: 10
posting:
  groups:
    - name: alt.binaries.test
      enabled: true
par2:
  enabled: true
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Par2.MaxConcurrentJobs != 1 {
		t.Errorf("Par2.MaxConcurrentJobs after Load = %d, want 1", cfg.Par2.MaxConcurrentJobs)
	}
	// 0-means-auto fields must stay 0 when omitted.
	if cfg.Posting.UploadBufferMemoryLimit != 0 {
		t.Errorf("Posting.UploadBufferMemoryLimit after Load = %d, want 0", cfg.Posting.UploadBufferMemoryLimit)
	}
	if cfg.PostCheck.MaxConcurrentChecks != 0 {
		t.Errorf("PostCheck.MaxConcurrentChecks after Load = %d, want 0", cfg.PostCheck.MaxConcurrentChecks)
	}
}

func TestValidate_Phase0NegativeValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ConfigData)
		wantErr bool
	}{
		{"valid zero values", func(c *ConfigData) {}, false},
		{"negative upload_buffer_memory_limit", func(c *ConfigData) {
			c.Posting.UploadBufferMemoryLimit = -1
		}, true},
		{"negative par2 max_concurrent_jobs", func(c *ConfigData) {
			c.Par2.MaxConcurrentJobs = -1
		}, true},
		{"negative post_check max_concurrent_checks", func(c *ConfigData) {
			c.PostCheck.MaxConcurrentChecks = -1
		}, true},
		{"positive values accepted", func(c *ConfigData) {
			c.Posting.UploadBufferMemoryLimit = 128 * 1024 * 1024
			c.Par2.MaxConcurrentJobs = 2
			c.PostCheck.MaxConcurrentChecks = 8
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validBaseConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
