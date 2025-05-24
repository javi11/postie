package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
servers:
  - host: news.example.com
    port: 119
    username: user
    password: pass
    ssl: true
    max_connections: 10
    insecure_ssl: false

posting:
  max_retries: 3
  retry_delay: 5s
  article_size_in_bytes: 750000
  groups:
    - alt.bin.test
  throttle_rate: 1048576
  message_id_format: random
  obfuscation_policy: full
  group_policy: each_file
  post_headers:
    add_ngx_header: false
    default_from: ''

post_check:
  enabled: true
  delay: 10s
  max_reposts: 1

par2:
  enabled: true
  redundancy: '1n*1.2'
  volume_size: 153600000
  max_input_slices: 4000
  extra_par2_options: []

nzb_compression:
  enabled: false
  type: zstd
  level: 5

watcher:
  size_threshold: 104857600
  schedule:
    start_time: '00:00'
    end_time: '23:59'
  ignore_patterns:
    - '*.tmp'
  min_file_size: 1048576
  check_interval: 5m
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test loading the config
	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Test all the getter methods
	t.Run("GetPostingConfig", func(t *testing.T) {
		postingCfg := cfg.GetPostingConfig()
		assert.Equal(t, 3, postingCfg.MaxRetries)
		assert.Equal(t, 5*time.Second, postingCfg.RetryDelay)
		assert.Equal(t, uint64(750000), postingCfg.ArticleSizeInBytes)
		assert.Equal(t, []string{"alt.bin.test"}, postingCfg.Groups)
		assert.Equal(t, int64(1048576), postingCfg.ThrottleRate)
		assert.Equal(t, MessageIDFormatRandom, postingCfg.MessageIDFormat)
		assert.Equal(t, ObfuscationPolicyFull, postingCfg.ObfuscationPolicy)
		assert.Equal(t, GroupPolicyEachFile, postingCfg.GroupPolicy)
		assert.False(t, postingCfg.PostHeaders.AddNGXHeader)
		assert.Equal(t, "", postingCfg.PostHeaders.DefaultFrom)
		assert.True(t, *postingCfg.WaitForPar2)
	})

	t.Run("GetPostCheckConfig", func(t *testing.T) {
		postCheckCfg := cfg.GetPostCheckConfig()
		assert.True(t, *postCheckCfg.Enabled)
		assert.Equal(t, 10*time.Second, postCheckCfg.RetryDelay)
		assert.Equal(t, uint(1), postCheckCfg.MaxRePost)
	})

	t.Run("GetPar2Config", func(t *testing.T) {
		ctx := context.Background()
		par2Cfg, err := cfg.GetPar2Config(ctx)
		require.NoError(t, err)
		assert.True(t, *par2Cfg.Enabled)
		assert.Equal(t, "1n*1.2", par2Cfg.Redundancy)
		assert.Equal(t, 153600000, par2Cfg.VolumeSize)
		assert.Equal(t, 4000, par2Cfg.MaxInputSlices)
		assert.Empty(t, par2Cfg.ExtraPar2Options)
	})

	t.Run("GetWatcherConfig", func(t *testing.T) {
		watcherCfg := cfg.GetWatcherConfig()
		assert.Equal(t, int64(104857600), watcherCfg.SizeThreshold)
		assert.Equal(t, "00:00", watcherCfg.Schedule.StartTime)
		assert.Equal(t, "23:59", watcherCfg.Schedule.EndTime)
		assert.Equal(t, []string{"*.tmp"}, watcherCfg.IgnorePatterns)
		assert.Equal(t, int64(1048576), watcherCfg.MinFileSize)
		assert.Equal(t, 5*time.Minute, watcherCfg.CheckInterval)
	})

	t.Run("GetNzbCompressionConfig", func(t *testing.T) {
		compressionCfg := cfg.GetNzbCompressionConfig()
		assert.False(t, compressionCfg.Enabled)
		assert.Equal(t, CompressionTypeZstd, compressionCfg.Type)
		assert.Equal(t, 5, compressionCfg.Level)
	})
}

func TestLoadWithInvalidConfig(t *testing.T) {
	// Test cases with invalid configurations
	testCases := []struct {
		name          string
		configContent string
		expectedErr   string
	}{
		{
			name: "no servers",
			configContent: `
posting:
  groups:
    - alt.bin.test
`,
			expectedErr: "invalid configuration: no servers configured",
		},
		{
			name: "server without host",
			configContent: `
servers:
  - port: 119
posting:
  groups:
    - alt.bin.test
`,
			expectedErr: "invalid configuration: server 0: host is required",
		},
		{
			name: "server with invalid port",
			configContent: `
servers:
  - host: news.example.com
    port: 0
posting:
  groups:
    - alt.bin.test
`,
			expectedErr: "invalid configuration: server 0: invalid port number",
		},
		{
			name: "server with invalid max connections",
			configContent: `
servers:
  - host: news.example.com
    port: 119
    max_connections: 0
posting:
  groups:
    - alt.bin.test
`,
			expectedErr: "invalid configuration: server 0: max_connections must be positive",
		},
		{
			name: "no posting groups",
			configContent: `
servers:
  - host: news.example.com
    port: 119
    max_connections: 10
posting:
  groups: []
`,
			expectedErr: "invalid configuration: posting groups are required",
		},
		{
			name: "invalid compression type",
			configContent: `
servers:
  - host: news.example.com
    port: 119
    max_connections: 10
posting:
  groups:
    - alt.bin.test
nzb_compression:
  enabled: true
  type: invalid
`,
			expectedErr: "invalid configuration: invalid compression type: invalid",
		},
		{
			name: "invalid zstd compression level",
			configContent: `
servers:
  - host: news.example.com
    port: 119
    max_connections: 10
posting:
  groups:
    - alt.bin.test
nzb_compression:
  enabled: true
  type: zstd
  level: 23
`,
			expectedErr: "invalid configuration: invalid zstd compression level: 23 (must be between 1-22)",
		},
		{
			name: "invalid brotli compression level",
			configContent: `
servers:
  - host: news.example.com
    port: 119
    max_connections: 10
posting:
  groups:
    - alt.bin.test
nzb_compression:
  enabled: true
  type: brotli
  level: 12
`,
			expectedErr: "invalid configuration: invalid brotli compression level: 12 (must be between 0-11)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tc.configContent), 0644)
			require.NoError(t, err)

			_, err = Load(configPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestLoadWithDefaults(t *testing.T) {
	// Test that default values are set correctly
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Minimal config with no explicit values for fields that have defaults
	configContent := `
servers:
  - host: news.example.com
    port: 119
    max_connections: 10
posting:
  groups:
    - alt.bin.test
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Check default values
	postingCfg := cfg.GetPostingConfig()
	assert.Equal(t, 3, postingCfg.MaxRetries)
	assert.Equal(t, 5*time.Second, postingCfg.RetryDelay)
	assert.Equal(t, uint64(750000), postingCfg.ArticleSizeInBytes)
	assert.Equal(t, int64(1024*1024), postingCfg.ThrottleRate)
	assert.Equal(t, GroupPolicyEachFile, postingCfg.GroupPolicy)
	assert.Equal(t, MessageIDFormatRandom, postingCfg.MessageIDFormat)
	assert.Equal(t, ObfuscationPolicyFull, postingCfg.ObfuscationPolicy)
	assert.True(t, *postingCfg.WaitForPar2)

	postCheckCfg := cfg.GetPostCheckConfig()
	assert.True(t, *postCheckCfg.Enabled)

	ctx := context.Background()
	par2Cfg, err := cfg.GetPar2Config(ctx)
	require.NoError(t, err)
	assert.True(t, *par2Cfg.Enabled)
	assert.Equal(t, defaultRedundancy, par2Cfg.Redundancy)
	assert.Equal(t, defaultVolumeSize, par2Cfg.VolumeSize)
	assert.Equal(t, defaultMaxInputSlices, par2Cfg.MaxInputSlices)

	compressionCfg := cfg.GetNzbCompressionConfig()
	assert.False(t, compressionCfg.Enabled)
	assert.Equal(t, CompressionTypeNone, compressionCfg.Type)
}

func TestGetNNTPPool(t *testing.T) {
	// Create a minimal config for testing GetNNTPPool
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
servers:
  - host: news.example.com
    port: 119
    username: user
    password: pass
    ssl: true
    max_connections: 10
posting:
  groups:
    - alt.bin.test
connection_pool:
  skip_providers_verification_on_creation: true
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// With skip_providers_verification_on_creation set to true, this should work
	// without attempting to connect to the server
	pool, err := cfg.GetNNTPPool()
	require.NoError(t, err)
	assert.NotNil(t, pool)
}

func TestNonExistentConfigFile(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error reading config file")
}

func TestInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	invalidYAML := `
servers:
  - host: "news.example.com
    port: 119
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing config file")
}

// TestEnsurePar2Executable tests the ensurePar2Executable function
func TestEnsurePar2Executable(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Test case 1: Par2 executable exists
	t.Run("Par2ExecutableExists", func(t *testing.T) {
		// Create a dummy executable file
		dummyExePath := filepath.Join(tempDir, "dummy_par2")
		err := os.WriteFile(dummyExePath, []byte("dummy content"), 0755)
		require.NoError(t, err)

		// Call ensurePar2Executable
		ctx := context.Background()
		resultPath, err := ensurePar2Executable(ctx, dummyExePath)
		require.NoError(t, err)
		assert.Equal(t, dummyExePath, resultPath)
	})

	// Test case 2: Par2 executable doesn't exist (this will trigger a download)
	// We can't easily test this without mocking the download function,
	// so we'll just verify that the function correctly identifies a non-existent file
	t.Run("Par2ExecutableDoesNotExist", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent_par2")

		// Verify the file doesn't exist
		_, err := os.Stat(nonExistentPath)
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))

		// We won't call ensurePar2Executable here because it would try to download
		// the actual executable, which we don't want in a unit test
	})
}

// TestValidate tests the validate method directly
func TestValidate(t *testing.T) {
	testCases := []struct {
		name        string
		config      *config
		expectedErr string
	}{
		{
			name: "valid configuration",
			config: &config{
				Servers: []ServerConfig{
					{
						Host:           "news.example.com",
						Port:           119,
						MaxConnections: 10,
					},
				},
				Posting: PostingConfig{
					Groups: []string{"alt.bin.test"},
				},
			},
			expectedErr: "",
		},
		{
			name: "no servers",
			config: &config{
				Posting: PostingConfig{
					Groups: []string{"alt.bin.test"},
				},
			},
			expectedErr: "no servers configured",
		},
		{
			name: "server without host",
			config: &config{
				Servers: []ServerConfig{
					{
						Port:           119,
						MaxConnections: 10,
					},
				},
				Posting: PostingConfig{
					Groups: []string{"alt.bin.test"},
				},
			},
			expectedErr: "server 0: host is required",
		},
		{
			name: "server with invalid port",
			config: &config{
				Servers: []ServerConfig{
					{
						Host:           "news.example.com",
						Port:           0,
						MaxConnections: 10,
					},
				},
				Posting: PostingConfig{
					Groups: []string{"alt.bin.test"},
				},
			},
			expectedErr: "server 0: invalid port number",
		},
		{
			name: "server with invalid max connections",
			config: &config{
				Servers: []ServerConfig{
					{
						Host: "news.example.com",
						Port: 119,
					},
				},
				Posting: PostingConfig{
					Groups: []string{"alt.bin.test"},
				},
			},
			expectedErr: "server 0: max_connections must be positive",
		},
		{
			name: "no posting groups",
			config: &config{
				Servers: []ServerConfig{
					{
						Host:           "news.example.com",
						Port:           119,
						MaxConnections: 10,
					},
				},
				Posting: PostingConfig{
					Groups: []string{},
				},
			},
			expectedErr: "posting groups are required",
		},
		{
			name: "invalid compression type",
			config: &config{
				Servers: []ServerConfig{
					{
						Host:           "news.example.com",
						Port:           119,
						MaxConnections: 10,
					},
				},
				Posting: PostingConfig{
					Groups: []string{"alt.bin.test"},
				},
				NzbCompression: NzbCompressionConfig{
					Enabled: true,
					Type:    "invalid",
				},
			},
			expectedErr: "invalid compression type: invalid",
		},
		{
			name: "invalid zstd compression level",
			config: &config{
				Servers: []ServerConfig{
					{
						Host:           "news.example.com",
						Port:           119,
						MaxConnections: 10,
					},
				},
				Posting: PostingConfig{
					Groups: []string{"alt.bin.test"},
				},
				NzbCompression: NzbCompressionConfig{
					Enabled: true,
					Type:    CompressionTypeZstd,
					Level:   23,
				},
			},
			expectedErr: "invalid zstd compression level: 23 (must be between 1-22)",
		},
		{
			name: "invalid brotli compression level",
			config: &config{
				Servers: []ServerConfig{
					{
						Host:           "news.example.com",
						Port:           119,
						MaxConnections: 10,
					},
				},
				Posting: PostingConfig{
					Groups: []string{"alt.bin.test"},
				},
				NzbCompression: NzbCompressionConfig{
					Enabled: true,
					Type:    CompressionTypeBrotli,
					Level:   12,
				},
			},
			expectedErr: "invalid brotli compression level: 12 (must be between 0-11)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()
			if tc.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}

// TestGetPar2ConfigError tests the error case in GetPar2Config
func TestGetPar2ConfigError(t *testing.T) {
	// Since we can't easily mock the ensurePar2Executable function directly,
	// we'll create a test that simulates a failure in the GetPar2Config function

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create a config file with a non-existent par2 path
	configContent := `
servers:
  - host: news.example.com
    port: 119
    max_connections: 10
posting:
  groups:
    - alt.bin.test
par2:
  enabled: true
  par2_path: "/non/existent/path/that/will/fail"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Create a test-specific context that we can cancel to prevent the actual download
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately to force an error

	// Call GetPar2Config and expect an error due to the cancelled context
	_, err = cfg.GetPar2Config(ctx)
	require.Error(t, err)
}
