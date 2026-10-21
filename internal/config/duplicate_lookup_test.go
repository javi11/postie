package config

import "testing"

func TestDuplicateLookupConfigValidation(t *testing.T) {
	for _, tt := range []struct {
		name, url string
		timeout   Duration
		valid     bool
	}{{"disabled", "", "", true}, {"enabled", "https://lookup.example/api", "30s", true}, {"bad URL", "file:///tmp/test", "30s", false}, {"missing URL", "", "30s", false}, {"bad timeout", "https://lookup.example/api", "bad", false}, {"unbounded timeout", "https://lookup.example/api", "10m", false}} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := GetDefaultConfig()
			cfg.Servers = []ServerConfig{{Host: "news.example", Port: 119, MaxConnections: 1, Role: ServerRoleUpload}}
			cfg.DuplicateLookup = DuplicateLookupConfig{Enabled: tt.name != "disabled", URL: tt.url, Timeout: tt.timeout}
			err := cfg.Validate()
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v: %v", tt.valid, err)
			}
		})
	}
}
