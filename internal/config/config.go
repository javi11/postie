package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/javi11/nntppool"
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

// Config represents the application configuration
type Config struct {
	Servers []ServerConfig `json:"servers"`
	Posting PostingConfig  `json:"posting"`
	// Check uploaded article configuration. used to check if an article was successfully uploaded and propagated.
	PostCheck PostCheck `json:"post_check"`
	// Path to the par2 executable. If not provided, the default cmdline par2 executable will be downloaded.
	Par2Exe string `json:"par2_exe"`
}

// ServerConfig represents a Usenet server configuration
type ServerConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	SSL            bool   `json:"ssl"`
	MaxConnections int    `json:"max_connections"`
}

type PostHeaders struct {
	// Whether to add the X-NXG header to the uploaded articles (You will still see this header in the generated NZB). Default value is `true`.
	// If obfuscation policy is `FULL` this header will not be added.
	// If message_id_format is not `ngx` this header will not be added.
	AddNGXHeader bool `json:"add_ngx_header"`
	// The default from header for the uploaded articles. By default a random poster will be used for each article. This will override GenerateFromByArticle
	DefaultFrom string `json:"default_from"`
	// Add custom headers to the uploaded articles. Subject, From, Newsgroups, Message-ID and Date can not be override.
	CustomHeaders []CustomHeader `json:"custom_headers"`
}

type CustomHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PostCheck struct {
	// If enabled articles will be checked after being posted. Default value is `true`.
	Enabled bool `json:"enabled"`
	// Delay between retries. Default value is `10s`.
	RetryDelay time.Duration `json:"delay"`
	// The maximum number of retries to check an article.
	MaxRetries uint `json:"max_retries"`
	// The maximum number of re-posts if article check fails. Default value is `1`.
	MaxRePost uint `json:"max_reposts"`
}

// PostingConfig represents posting configuration
type PostingConfig struct {
	MaxRetries         int             `json:"max_retries"`
	RetryDelay         time.Duration   `json:"retry_delay"`
	CheckInterval      time.Duration   `json:"check_interval"`
	MaxCheckRetries    int             `json:"max_check_retries"`
	ArticleSizeInBytes int64           `json:"article_size_in_bytes"`
	Groups             []string        `json:"groups"`
	ThrottleRate       int64           `json:"throttle_rate"` // bytes per second
	MaxWorkers         int             `json:"max_workers"`
	MessageIDFormat    MessageIDFormat `json:"message_id_format"`
	PostHeaders        PostHeaders     `json:"post_headers"`
	// If true the uploaded subject and filename will be obfuscated. Default value is `true`.
	ObfuscationPolicy ObfuscationPolicy `json:"obfuscation_policy"`
	//  If you give several Groups you've 3 policy when posting
	GroupPolicy GroupPolicy `json:"group_policy"`
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Set default values
	if cfg.Posting.MaxRetries <= 0 {
		cfg.Posting.MaxRetries = 3
	}
	if cfg.Posting.RetryDelay <= 0 {
		cfg.Posting.RetryDelay = 5 * time.Second
	}
	if cfg.Posting.CheckInterval <= 0 {
		cfg.Posting.CheckInterval = 1 * time.Minute
	}
	if cfg.Posting.MaxCheckRetries <= 0 {
		cfg.Posting.MaxCheckRetries = 3
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
func (c *Config) validate() error {
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
func (c *Config) GetNNTPPool() (nntppool.UsenetConnectionPool, error) {
	providers := make([]nntppool.UsenetProviderConfig, len(c.Servers))
	for i, s := range c.Servers {
		maxConnections := s.MaxConnections
		if maxConnections <= 0 {
			maxConnections = 10 // default value if not specified
		}
		providers[i] = nntppool.UsenetProviderConfig{
			Host:                           s.Host,
			Port:                           s.Port,
			Username:                       s.Username,
			Password:                       s.Password,
			TLS:                            s.SSL,
			MaxConnections:                 maxConnections,
			MaxConnectionIdleTimeInSeconds: 300,
			MaxConnectionTTLInSeconds:      3600,
			InsecureSSL:                    true,
		}
	}

	config := nntppool.Config{
		Providers:                           providers,
		HealthCheckInterval:                 time.Minute,
		MinConnections:                      5,
		MaxRetries:                          3,
		SkipProvidersVerificationOnCreation: true,
	}

	pool, err := nntppool.NewConnectionPool(config)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	return pool, nil
}
