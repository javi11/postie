package config

import (
	"context"
	"encoding/json"
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

// Duration wraps time.Duration to provide custom JSON and YAML marshalling
type Duration string

// MarshalJSON implements json.Marshaler interface
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(d))
}

// UnmarshalJSON implements json.Unmarshaler interface
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

// MarshalYAML implements yaml.Marshaler interface
func (d Duration) MarshalYAML() (interface{}, error) {
	return string(d), nil
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

// ToDuration converts Duration to time.Duration
func (d Duration) ToDuration() time.Duration {
	duration, _ := time.ParseDuration(string(d))
	return duration
}

type CompressionType string

const (
	// No compression
	CompressionTypeNone CompressionType = "none"
	// Zstandard compression
	CompressionTypeZstd CompressionType = "zstd"
	// Brotli compression
	CompressionTypeBrotli CompressionType = "brotli"
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
	// NXG: the Message-ID will be formatted as https://github.com/javi11/nxg
	MessageIDFormatNXG MessageIDFormat = "nxg"
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
	// - NXG-header: will not be added
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
	GetNzbCompressionConfig() NzbCompressionConfig
	GetQueueConfig() QueueConfig
	GetPostUploadScriptConfig() PostUploadScriptConfig
	GetMaintainOriginalExtension() bool
}

type ConnectionPoolConfig struct {
	MinConnections                      int      `yaml:"min_connections" json:"min_connections"`
	HealthCheckInterval                 Duration `yaml:"health_check_interval" json:"health_check_interval"`
	SkipProvidersVerificationOnCreation bool     `yaml:"skip_providers_verification_on_creation" json:"skip_providers_verification_on_creation"`
}

// config is the internal implementation of the Config interface
type ConfigData struct {
	Servers        []ServerConfig       `yaml:"servers" json:"servers"`
	ConnectionPool ConnectionPoolConfig `yaml:"connection_pool" json:"connection_pool"`
	Posting        PostingConfig        `yaml:"posting" json:"posting"`
	// Check uploaded article configuration. used to check if an article was successfully uploaded and propagated.
	PostCheck                 PostCheck              `yaml:"post_check" json:"post_check"`
	Par2                      Par2Config             `yaml:"par2" json:"par2"`
	Watcher                   WatcherConfig          `yaml:"watcher" json:"watcher"`
	NzbCompression            NzbCompressionConfig   `yaml:"nzb_compression" json:"nzb_compression"`
	Queue                     QueueConfig            `yaml:"queue" json:"queue"`
	OutputDir                 string                 `yaml:"output_dir" json:"output_dir"`
	MaintainOriginalExtension *bool                  `yaml:"maintain_original_extension" json:"maintain_original_extension"`
	PostUploadScript          PostUploadScriptConfig `yaml:"post_upload_script" json:"post_upload_script"`
}

type Par2Config struct {
	Enabled          *bool     `yaml:"enabled" json:"enabled"`
	Par2Path         string    `yaml:"par2_path" json:"par2_path"`
	Redundancy       string    `yaml:"redundancy" json:"redundancy"`
	VolumeSize       int       `yaml:"volume_size" json:"volume_size"`
	MaxInputSlices   int       `yaml:"max_input_slices" json:"max_input_slices"`
	ExtraPar2Options []string  `yaml:"extra_par2_options" json:"extra_par2_options"`
	TempDir          string    `yaml:"temp_dir" json:"temp_dir"`
	once             sync.Once `json:"-"`
}

// ServerConfig represents a Usenet server configuration
type ServerConfig struct {
	Host                           string `yaml:"host" json:"host"`
	Port                           int    `yaml:"port" json:"port"`
	Username                       string `yaml:"username" json:"username"`
	Password                       string `yaml:"password" json:"password"`
	SSL                            bool   `yaml:"ssl" json:"ssl"`
	MaxConnections                 int    `yaml:"max_connections" json:"max_connections"`
	MaxConnectionIdleTimeInSeconds int    `yaml:"max_connection_idle_time_in_seconds" json:"max_connection_idle_time_in_seconds"`
	MaxConnectionTTLInSeconds      int    `yaml:"max_connection_ttl_in_seconds" json:"max_connection_ttl_in_seconds"`
	InsecureSSL                    bool   `yaml:"insecure_ssl" json:"insecure_ssl"`
	Enabled                        *bool  `yaml:"enabled" json:"enabled"`
}

type PostHeaders struct {
	// Whether to add the X-NXG header to the uploaded articles (You will still see this header in the generated NZB). Default value is `true`.
	// If obfuscation policy is `FULL` this header will not be added.
	// If message_id_format is not `nxg` this header will not be added.
	AddNXGHeader bool `yaml:"add_nxg_header" json:"add_nxg_header"`
	// The default from header for the uploaded articles. By default a random poster will be used for each article. This will override GenerateFromByArticle
	DefaultFrom string `yaml:"default_from" json:"default_from"`
	// Add custom headers to the uploaded articles. Subject, From, Newsgroups, Message-ID and Date can not be override.
	CustomHeaders []CustomHeader `yaml:"custom_headers" json:"custom_headers"`
}

type CustomHeader struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

type PostCheck struct {
	// If enabled articles will be checked after being posted. Default value is `true`.
	Enabled *bool `yaml:"enabled" json:"enabled"`
	// Delay between retries. Default value is `10s`.
	RetryDelay Duration `yaml:"delay" json:"delay"`
	// The maximum number of re-posts if article check fails. Default value is `1`.
	MaxRePost uint `yaml:"max_reposts" json:"max_reposts"`
}

// PostingConfig represents posting configuration
type PostingConfig struct {
	WaitForPar2        *bool           `yaml:"wait_for_par2" json:"wait_for_par2"`
	MaxRetries         int             `yaml:"max_retries" json:"max_retries"`
	RetryDelay         Duration        `yaml:"retry_delay" json:"retry_delay"`
	ArticleSizeInBytes uint64          `yaml:"article_size_in_bytes" json:"article_size_in_bytes"`
	Groups             []string        `yaml:"groups" json:"groups"`
	ThrottleRate       int64           `yaml:"throttle_rate" json:"throttle_rate"` // bytes per second
	MessageIDFormat    MessageIDFormat `yaml:"message_id_format" json:"message_id_format"`
	PostHeaders        PostHeaders     `yaml:"post_headers" json:"post_headers"`
	// If true the uploaded subject and filename will be obfuscated. Default value is `true`.
	ObfuscationPolicy     ObfuscationPolicy `yaml:"obfuscation_policy" json:"obfuscation_policy"`
	Par2ObfuscationPolicy ObfuscationPolicy `yaml:"par2_obfuscation_policy" json:"par2_obfuscation_policy"`
	//  If you give several Groups you've 3 policy when posting
	GroupPolicy GroupPolicy `yaml:"group_policy" json:"group_policy"`
}

type WatcherConfig struct {
	Enabled            bool           `yaml:"enabled" json:"enabled"`
	WatchDirectory     string         `yaml:"watch_directory" json:"watch_directory"`
	SizeThreshold      int64          `yaml:"size_threshold" json:"size_threshold"`
	Schedule           ScheduleConfig `yaml:"schedule" json:"schedule"`
	IgnorePatterns     []string       `yaml:"ignore_patterns" json:"ignore_patterns"`
	MinFileSize        int64          `yaml:"min_file_size" json:"min_file_size"`
	CheckInterval      Duration       `yaml:"check_interval" json:"check_interval"`
	DeleteOriginalFile bool           `yaml:"delete_original_file" json:"delete_original_file"`
}

type ScheduleConfig struct {
	StartTime string `yaml:"start_time" json:"start_time"`
	EndTime   string `yaml:"end_time" json:"end_time"`
}

// NzbCompressionConfig represents the NZB compression configuration
type NzbCompressionConfig struct {
	// Whether to enable compression. Default is false.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Compression type to use. Default is "none".
	Type CompressionType `yaml:"type" json:"type"`
	// Compression level to use. Default depends on the compression type.
	Level int `yaml:"level" json:"level"`
}

// QueueConfig represents the upload queue configuration
type QueueConfig struct {
	// Database type to use for the queue. Supported: "sqlite", "postgres", "mysql"
	DatabaseType string `yaml:"database_type" json:"database_type"`
	// Database connection string or file path
	DatabasePath string `yaml:"database_path" json:"database_path"`
	// Maximum concurrent uploads from queue
	MaxConcurrentUploads int `yaml:"max_concurrent_uploads" json:"max_concurrent_uploads"`
}

// PostUploadScriptConfig represents the post upload script configuration
type PostUploadScriptConfig struct {
	// Whether to enable the post upload script execution. Default value is `false`.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Command to execute after NZB generation. Use {{nzb_path}} placeholder for the NZB file path
	Command string `yaml:"command" json:"command"`
	// Timeout for script execution. Default value is `30s`.
	Timeout Duration `yaml:"timeout" json:"timeout"`
}

// Par2DownloadStatus represents the status of par2 executable download
type Par2DownloadStatus struct {
	Status  string `json:"status"`  // "downloading", "completed", "error"
	Message string `json:"message"` // Human readable message
}

// ProgressStatus represents the progress of file processing operations
type ProgressStatus struct {
	CurrentFile         string  `json:"currentFile"`
	TotalFiles          int     `json:"totalFiles"`
	CompletedFiles      int     `json:"completedFiles"`
	Stage               string  `json:"stage"`
	Details             string  `json:"details"`
	IsRunning           bool    `json:"isRunning"`
	LastUpdate          int64   `json:"lastUpdate"`
	Percentage          float64 `json:"percentage"`
	CurrentFileProgress float64 `json:"currentFileProgress"`
	JobID               string  `json:"jobID"`
	TotalBytes          int64   `json:"totalBytes"`
	TransferredBytes    int64   `json:"transferredBytes"`
	CurrentFileBytes    int64   `json:"currentFileBytes"`
	Speed               float64 `json:"speed"`
	SecondsLeft         float64 `json:"secondsLeft"`
	ElapsedTime         float64 `json:"elapsedTime"`
}

// Load loads configuration from a file
func Load(path string) (*ConfigData, error) {
	enabled := true

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg ConfigData
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Set default values
	if cfg.Posting.MaxRetries <= 0 {
		cfg.Posting.MaxRetries = 3
	}

	if cfg.Posting.RetryDelay == "" {
		cfg.Posting.RetryDelay = Duration(5 * time.Second)
	}

	if cfg.Posting.ArticleSizeInBytes <= 0 {
		cfg.Posting.ArticleSizeInBytes = 750000 // Default to 750KB
	}

	if cfg.Posting.ThrottleRate <= 0 {
		cfg.Posting.ThrottleRate = 0 // Default to unlimited
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

	if cfg.Par2.Enabled == nil {
		cfg.Par2.Enabled = &enabled
	}

	if cfg.Posting.WaitForPar2 == nil {
		cfg.Posting.WaitForPar2 = &enabled
	} else if !*cfg.Posting.WaitForPar2 {
		if !*cfg.Par2.Enabled {
			cfg.Posting.WaitForPar2 = &enabled
		} else {
			slog.Warn("Use it at your own risk. Par2 files will be created and uploaded in parallel with the original file, " +
				"if par2 creation fails or posting fails, you will end up uploading something that is trash.")
		}
	}

	if cfg.PostCheck.Enabled == nil {
		cfg.PostCheck.Enabled = &enabled
	}

	if cfg.Par2.Par2Path == "" {
		cfg.Par2.Par2Path = "./" + filepath.Join(filepath.Dir(path), defaultPar2Path)
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

	// Set default values for NZB compression
	if cfg.NzbCompression.Type == "" {
		cfg.NzbCompression.Type = CompressionTypeNone
	}

	// Set default compression level based on type
	if cfg.NzbCompression.Level == 0 {
		switch cfg.NzbCompression.Type {
		case CompressionTypeZstd:
			cfg.NzbCompression.Level = 3 // Default zstd level
		case CompressionTypeBrotli:
			cfg.NzbCompression.Level = 4 // Default brotli level
		}
	}

	// Set default values for Queue configuration
	if cfg.Queue.DatabaseType == "" {
		cfg.Queue.DatabaseType = "sqlite"
	}

	if cfg.Queue.DatabasePath == "" {
		cfg.Queue.DatabasePath = "./postie_queue.db"
	}

	if cfg.Queue.MaxConcurrentUploads <= 0 {
		cfg.Queue.MaxConcurrentUploads = 1
	}

	// Set default for maintain original extension (default to true)
	if cfg.MaintainOriginalExtension == nil {
		cfg.MaintainOriginalExtension = &enabled
	}

	// Set default enabled state for servers
	for i := range cfg.Servers {
		if cfg.Servers[i].Enabled == nil {
			cfg.Servers[i].Enabled = &enabled
		}
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validate validates the configuration
func (c *ConfigData) validate() error {
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

	// Validate compression configuration
	if c.NzbCompression.Enabled {
		switch c.NzbCompression.Type {
		case CompressionTypeZstd:
			// zstd levels are between 1-22
			if c.NzbCompression.Level < 1 || c.NzbCompression.Level > 22 {
				return fmt.Errorf("invalid zstd compression level: %d (must be between 1-22)", c.NzbCompression.Level)
			}
		case CompressionTypeBrotli:
			// brotli levels are between 0-11
			if c.NzbCompression.Level < 0 || c.NzbCompression.Level > 11 {
				return fmt.Errorf("invalid brotli compression level: %d (must be between 0-11)", c.NzbCompression.Level)
			}
		case CompressionTypeNone:
			// Do nothing
		default:
			return fmt.Errorf("invalid compression type: %s", c.NzbCompression.Type)
		}
	}

	// Validate queue configuration
	switch c.Queue.DatabaseType {
	case "sqlite", "postgres", "mysql":
		// Valid database types
	default:
		return fmt.Errorf("invalid queue database type: %s (supported: sqlite, postgres, mysql)", c.Queue.DatabaseType)
	}

	if c.Queue.MaxConcurrentUploads <= 0 {
		return fmt.Errorf("queue max concurrent uploads must be positive")
	}

	return nil
}

// GetNNTPPool returns the NNTP connection pool
func (c *ConfigData) GetNNTPPool() (nntppool.UsenetConnectionPool, error) {
	// Filter enabled servers
	var enabledServers []ServerConfig
	for _, s := range c.Servers {
		if s.Enabled == nil || *s.Enabled {
			enabledServers = append(enabledServers, s)
		}
	}

	providers := make([]nntppool.UsenetProviderConfig, len(enabledServers))
	for i, s := range enabledServers {
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

	if c.ConnectionPool.HealthCheckInterval == "" {
		c.ConnectionPool.HealthCheckInterval = Duration(time.Minute)
	}

	if c.ConnectionPool.MinConnections <= 0 {
		c.ConnectionPool.MinConnections = 5
	}

	config := nntppool.Config{
		Providers:                           providers,
		HealthCheckInterval:                 c.ConnectionPool.HealthCheckInterval.ToDuration(),
		MinConnections:                      c.ConnectionPool.MinConnections,
		MaxRetries:                          uint(c.Posting.MaxRetries),
		DelayType:                           nntppool.DelayTypeExponential,
		RetryDelay:                          c.Posting.RetryDelay.ToDuration(),
		SkipProvidersVerificationOnCreation: c.ConnectionPool.SkipProvidersVerificationOnCreation,
	}

	pool, err := nntppool.NewConnectionPool(config)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	return pool, nil
}

func (c *ConfigData) GetPar2Config(ctx context.Context) (*Par2Config, error) {
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

func (c *ConfigData) GetPostingConfig() PostingConfig {
	return c.Posting
}

func (c *ConfigData) GetPostCheckConfig() PostCheck {
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

func (c *ConfigData) GetWatcherConfig() WatcherConfig {
	return c.Watcher
}

func (c *ConfigData) GetNzbCompressionConfig() NzbCompressionConfig {
	return c.NzbCompression
}

func (c *ConfigData) GetQueueConfig() QueueConfig {
	return c.Queue
}

func (c *ConfigData) GetOutputDir() string {
	if c.OutputDir != "" {
		return c.OutputDir
	}

	// Default to "./output" if not configured
	return "./output"
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() ConfigData {
	enabled := true
	disabled := false
	return ConfigData{
		Servers: []ServerConfig{},
		ConnectionPool: ConnectionPoolConfig{
			MinConnections:                      5,
			HealthCheckInterval:                 Duration(time.Minute),
			SkipProvidersVerificationOnCreation: false,
		},
		Posting: PostingConfig{
			WaitForPar2:        &enabled,
			MaxRetries:         3,
			RetryDelay:         Duration(5 * time.Second),
			ArticleSizeInBytes: 750000, // 750KB
			Groups:             []string{"alt.binaries.test"},
			ThrottleRate:       1048576, // 1MB/s
			MessageIDFormat:    MessageIDFormatRandom,
			PostHeaders: PostHeaders{
				AddNXGHeader:  false,
				DefaultFrom:   "",
				CustomHeaders: []CustomHeader{},
			},
			ObfuscationPolicy:     ObfuscationPolicyFull,
			Par2ObfuscationPolicy: ObfuscationPolicyFull,
			GroupPolicy:           GroupPolicyEachFile,
		},
		PostCheck: PostCheck{
			Enabled:    &enabled,
			RetryDelay: Duration(10 * time.Second),
			MaxRePost:  1,
		},
		Par2: Par2Config{
			Enabled:          &enabled,
			Par2Path:         defaultPar2Path,
			Redundancy:       defaultRedundancy,
			VolumeSize:       defaultVolumeSize,
			MaxInputSlices:   defaultMaxInputSlices,
			ExtraPar2Options: []string{},
			TempDir:          os.TempDir(),
		},
		Watcher: WatcherConfig{
			Enabled:        false,
			WatchDirectory: "",        // Will be set to default in backend if empty
			SizeThreshold:  104857600, // 100MB
			Schedule: ScheduleConfig{
				StartTime: "00:00",
				EndTime:   "23:59",
			},
			IgnorePatterns:     []string{"*.tmp", "*.part", "*.!ut"},
			MinFileSize:        1048576, // 1MB
			CheckInterval:      Duration("5m"),
			DeleteOriginalFile: false, // Default to keeping original files for safety
		},
		NzbCompression: NzbCompressionConfig{
			Enabled: disabled,
			Type:    CompressionTypeNone,
			Level:   0,
		},
		Queue: QueueConfig{
			DatabaseType:         "sqlite",
			DatabasePath:         "./postie_queue.db",
			MaxConcurrentUploads: 1,
		},
		OutputDir:                 "./output",
		MaintainOriginalExtension: &enabled,
		PostUploadScript: PostUploadScriptConfig{
			Enabled: false,
			Command: "",
			Timeout: Duration("30s"),
		},
	}
}

// SaveConfig saves a ConfigData to a file
func SaveConfig(configData *ConfigData, path string) error {
	data, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

func (c *ConfigData) GetPostUploadScriptConfig() PostUploadScriptConfig {
	return c.PostUploadScript
}

func (c *ConfigData) GetMaintainOriginalExtension() bool {
	if c.MaintainOriginalExtension == nil {
		return true // Default to true
	}
	return *c.MaintainOriginalExtension
}
