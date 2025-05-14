package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/javi11/nntppool"
	"github.com/javi11/postie/pkg/parpardownloader"
	"gopkg.in/yaml.v3"
)

const (
	defaultPar2Path       = "./parpar"
	defaultVolumeSize     = 153600000 // 150MB
	defaultRedundancy     = "1n*1.2"  //https://github.com/animetosho/ParPar/blob/6feee4dd94bb18480f0bf08cd9d17ffc7e671b69/help-full.txt#L75
	defaultMaxInputSlices = 4000
)

type GroupPolicy string

const (
	//    ALL       : everything is posted on ALL the Groups
	GroupPolicyAll GroupPolicy = "all"
	//    EACH_FILE : each File will be posted on a random Group from the list (only with Article's obfuscation)
	GroupPolicyEachFile GroupPolicy = "each_file"
)

type MessageIDFormat string

const (
	// NGX: the Message-ID will be formatted as https://github.com/javi11/nxg
	MessageIDFormatNGX MessageIDFormat = "ngx"
	// Random: the Message-ID will be a random string of 32 characters
	MessageIDFormatRandom MessageIDFormat = "random"
)

type ObfuscationPolicy string

const (
	// Will do the following obfuscation:
	// - Subject: will be obfuscated
	// - Filename: will be obfuscated
	// - Yenc header filename: will be randomized for every article
	// - Date: will be randomized for every article within last 6 hours
	// - NGX-header: will not be added
	// - Poster: will be random for each article
	ObfuscationPolicyFull ObfuscationPolicy = "full"
	// Will do the following obfuscation:
	// - Subject: will be obfuscated
	// - Filename: will be obfuscated
	// - Yenc header filename: will be same one for all articles
	// - Date: will be the real posted date
	// - Poster: will be the same one for all articles
	ObfuscationPolicyPartial ObfuscationPolicy = "partial"
	// Nothing will be obfuscated
	ObfuscationPolicyNone ObfuscationPolicy = "none"
)

type Config interface {
	GetNNTPPool() (nntppool.UsenetConnectionPool, error)
	GetPostingConfig() PostingConfig
	GetPostCheckConfig() PostCheck
	GetPar2Config(ctx context.Context) (*Par2Config, error)
	GetWatcherConfig() WatcherConfig
}

type ConnectionPoolConfig struct {
	MinConnections                      int           `yaml:"min_connections"`
	HealthCheckInterval                 time.Duration `yaml:"health_check_interval"`
	SkipProvidersVerificationOnCreation bool          `yaml:"skip_providers_verification_on_creation"`
}

// Config represents the application configuration
type config struct {
	Servers        []ServerConfig       `yaml:"servers"`
	ConnectionPool ConnectionPoolConfig `yaml:"connection_pool"`
	Posting        PostingConfig        `yaml:"posting"`
	// Check uploaded article configuration. used to check if an article was successfully uploaded and propagated.
	PostCheck PostCheck     `yaml:"post_check"`
	Par2      Par2Config    `yaml:"par2"`
	Watcher   WatcherConfig `yaml:"watcher"`
}

type Par2Config struct {
	Par2Path         string   `yaml:"par2_path"`
	Redundancy       string   `yaml:"redundancy"`
	VolumeSize       int      `yaml:"volume_size"`
	MaxInputSlices   int      `yaml:"max_input_slices"`
	ExtraPar2Options []string `yaml:"extra_par2_options"`
	once             sync.Once
}

// ServerConfig represents a Usenet server configuration
type ServerConfig struct {
	Host                           string `yaml:"host"`
	Port                           int    `yaml:"port"`
	Username                       string `yaml:"username"`
	Password                       string `yaml:"password"`
	SSL                            bool   `yaml:"ssl"`
	MaxConnections                 int    `yaml:"max_connections"`
	MaxConnectionIdleTimeInSeconds int    `yaml:"max_connection_idle_time_in_seconds"`
	MaxConnectionTTLInSeconds      int    `yaml:"max_connection_ttl_in_seconds"`
	InsecureSSL                    bool   `yaml:"insecure_ssl"`
}

type PostHeaders struct {
	// Whether to add the X-NXG header to the uploaded articles (You will still see this header in the generated NZB). Default value is `true`.
	// If obfuscation policy is `FULL` this header will not be added.
	// If message_id_format is not `ngx` this header will not be added.
	AddNGXHeader bool `yaml:"add_ngx_header"`
	// The default from header for the uploaded articles. By default a random poster will be used for each article. This will override GenerateFromByArticle
	DefaultFrom string `yaml:"default_from"`
	// Add custom headers to the uploaded articles. Subject, From, Newsgroups, Message-ID and Date can not be override.
	CustomHeaders []CustomHeader `yaml:"custom_headers"`
}

type CustomHeader struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type PostCheck struct {
	// If enabled articles will be checked after being posted. Default value is `true`.
	Enabled bool `yaml:"enabled"`
	// Delay between retries. Default value is `10s`.
	RetryDelay time.Duration `yaml:"delay"`
	// The maximum number of re-posts if article check fails. Default value is `1`.
	MaxRePost uint `yaml:"max_reposts"`
}

// PostingConfig represents posting configuration
type PostingConfig struct {
	MaxRetries         int             `yaml:"max_retries"`
	RetryDelay         time.Duration   `yaml:"retry_delay"`
	ArticleSizeInBytes uint64          `yaml:"article_size_in_bytes"`
	Groups             []string        `yaml:"groups"`
	ThrottleRate       int64           `yaml:"throttle_rate"` // bytes per second
	MaxWorkers         int             `yaml:"max_workers"`
	MessageIDFormat    MessageIDFormat `yaml:"message_id_format"`
	PostHeaders        PostHeaders     `yaml:"post_headers"`
	// If true the uploaded subject and filename will be obfuscated. Default value is `true`.
	ObfuscationPolicy     ObfuscationPolicy `yaml:"obfuscation_policy"`
	Par2ObfuscationPolicy ObfuscationPolicy `yaml:"par2_obfuscation_policy"`
	//  If you give several Groups you've 3 policy when posting
	GroupPolicy GroupPolicy `yaml:"group_policy"`
}

type WatcherConfig struct {
	SizeThreshold  int64          `yaml:"size_threshold"`
	Schedule       ScheduleConfig `yaml:"schedule"`
	IgnorePatterns []string       `yaml:"ignore_patterns"`
	MinFileSize    int64          `yaml:"min_file_size"`
	CheckInterval  time.Duration  `yaml:"check_interval"`
}

type ScheduleConfig struct {
	StartTime string `yaml:"start_time"`
	EndTime   string `yaml:"end_time"`
}

// Load loads configuration from a file
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Set default values
	if cfg.Posting.MaxRetries <= 0 {
		cfg.Posting.MaxRetries = 3
	}

	if cfg.Posting.RetryDelay <= 0 {
		cfg.Posting.RetryDelay = 5 * time.Second
	}

	if cfg.Posting.ArticleSizeInBytes <= 0 {
		cfg.Posting.ArticleSizeInBytes = 750000 // Default to 750KB
	}

	if cfg.Posting.ThrottleRate <= 0 {
		cfg.Posting.ThrottleRate = 1024 * 1024 // Default to 1MB/s
	}

	if cfg.Posting.GroupPolicy == "" {
		cfg.Posting.GroupPolicy = GroupPolicyEachFile
	}

	if cfg.Posting.MessageIDFormat == "" {
		cfg.Posting.MessageIDFormat = MessageIDFormatRandom
	}

	if cfg.Posting.ObfuscationPolicy == "" {
		cfg.Posting.ObfuscationPolicy = ObfuscationPolicyFull
	}

	if cfg.Par2.Par2Path == "" {
		cfg.Par2.Par2Path = filepath.Join(filepath.Dir(path), "./parpar")
	}

	if cfg.Par2.VolumeSize <= 0 {
		cfg.Par2.VolumeSize = defaultVolumeSize
	}

	if cfg.Par2.Redundancy == "" {
		cfg.Par2.Redundancy = defaultRedundancy
	}

	if cfg.Par2.MaxInputSlices <= 0 {
		cfg.Par2.MaxInputSlices = defaultMaxInputSlices
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	maxWorkers := 0

	for _, s := range cfg.Servers {
		maxWorkers += s.MaxConnections
	}

	cfg.Posting.MaxWorkers = maxWorkers

	return &cfg, nil
}

// validate validates the configuration
func (c *config) validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	for i, s := range c.Servers {
		if s.Host == "" {
			return fmt.Errorf("server %d: host is required", i)
		}

		if s.Port <= 0 || s.Port > 65535 {
			return fmt.Errorf("server %d: invalid port number", i)
		}

		if s.MaxConnections <= 0 {
			return fmt.Errorf("server %d: max_connections must be positive", i)
		}
	}

	if len(c.Posting.Groups) == 0 {
		return fmt.Errorf("posting groups are required")
	}

	return nil
}

// GetNNTPPool returns the NNTP connection pool
func (c *config) GetNNTPPool() (nntppool.UsenetConnectionPool, error) {
	providers := make([]nntppool.UsenetProviderConfig, len(c.Servers))
	for i, s := range c.Servers {
		maxConnections := s.MaxConnections
		if maxConnections <= 0 {
			maxConnections = 10 // default value if not specified
		}

		if s.MaxConnectionIdleTimeInSeconds <= 0 {
			s.MaxConnectionIdleTimeInSeconds = 300
		}

		if s.MaxConnectionTTLInSeconds <= 0 {
			s.MaxConnectionTTLInSeconds = 3600
		}

		providers[i] = nntppool.UsenetProviderConfig{
			Host:                           s.Host,
			Port:                           s.Port,
			Username:                       s.Username,
			Password:                       s.Password,
			TLS:                            s.SSL,
			MaxConnections:                 maxConnections,
			MaxConnectionIdleTimeInSeconds: s.MaxConnectionIdleTimeInSeconds,
			MaxConnectionTTLInSeconds:      s.MaxConnectionTTLInSeconds,
			InsecureSSL:                    s.InsecureSSL,
		}
	}

	if c.ConnectionPool.HealthCheckInterval <= 0 {
		c.ConnectionPool.HealthCheckInterval = time.Minute
	}

	if c.ConnectionPool.MinConnections <= 0 {
		c.ConnectionPool.MinConnections = 5
	}

	config := nntppool.Config{
		Providers:                           providers,
		HealthCheckInterval:                 c.ConnectionPool.HealthCheckInterval,
		MinConnections:                      c.ConnectionPool.MinConnections,
		MaxRetries:                          uint(c.Posting.MaxRetries),
		DelayType:                           nntppool.DelayTypeExponential,
		RetryDelay:                          c.Posting.RetryDelay,
		SkipProvidersVerificationOnCreation: c.ConnectionPool.SkipProvidersVerificationOnCreation,
	}

	pool, err := nntppool.NewConnectionPool(config)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	return pool, nil
}

func (c *config) GetPar2Config(ctx context.Context) (*Par2Config, error) {
	var errDownload error
	c.Par2.once.Do(func() {
		par2ExePath, err := ensurePar2Executable(ctx, c.Par2.Par2Path)
		if err != nil {
			slog.ErrorContext(ctx, "Error ensuring par2 executable", "error", err)
			errDownload = fmt.Errorf("error ensuring par2 executable: %w", err)

			return
		}

		c.Par2.Par2Path = par2ExePath
	})

	if errDownload != nil {
		return nil, errDownload
	}

	return &c.Par2, nil
}

func (c *config) GetPostingConfig() PostingConfig {
	return c.Posting
}

func (c *config) GetPostCheckConfig() PostCheck {
	return c.PostCheck
}

// ensurePar2Executable checks if a par2 executable is configured, downloads one if necessary,
// and returns the final path to the executable.
func ensurePar2Executable(ctx context.Context, par2Path string) (string, error) {
	slog.DebugContext(ctx, "Using configured Par2 executable", "path", par2Path)
	// Verify it exists?
	if _, err := os.Stat(par2Path); err == nil {
		return par2Path, nil
	}

	slog.WarnContext(ctx, "Configured Par2 executable not found, proceeding to download", "path", par2Path)

	// Download if not configured and not found in default path
	slog.InfoContext(ctx, "No par2 executable configured or found, downloading parpar...")
	execPath, err := parpardownloader.DownloadParParCmd(par2Path)
	if err != nil {
		return "", fmt.Errorf("failed to download parpar: %w", err)
	}

	slog.InfoContext(ctx, "Downloaded Par2 executable", "path", execPath)

	return execPath, nil
}

func (c *config) GetWatcherConfig() WatcherConfig {
	return c.Watcher
}
