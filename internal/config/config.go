// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the DNS server
type Config struct {
	// DNS Server settings
	DNSPort string

	// Database configuration
	Database DatabaseConfig

	// Cache configuration
	Cache CacheConfig

	// Redis configuration
	Redis RedisConfig

	// Priority configuration
	Priority PriorityConfig

	// Server behavior
	MaxConcurrentQueries int
	ShutdownTimeout      time.Duration

	// Logging configuration
	Logging LoggingConfig

	// Logging
	LogLevel string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level           string  `json:"level"`
	Directory       string  `json:"directory"`
	AppLogFile      string  `json:"app_log_file"`
	QueryLogFile    string  `json:"query_log_file"`
	ErrorLogFile    string  `json:"error_log_file"`
	EnableConsole   bool    `json:"enable_console"`
	QuerySampleRate float64 `json:"query_sample_rate"`
	BufferSize      int     `json:"buffer_size"`
}

// DatabaseConfig holds PostgreSQL database configuration
type DatabaseConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	DBName         string
	SSLMode        string
	ConnectionName string

	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled         bool
	MaxEntries      int
	CleanupInterval time.Duration
	DefaultTTL      time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Enabled         bool          `json:"enabled"`
	Address         string        `json:"address"`
	Password        string        `json:"password"`
	Database        int           `json:"database"`
	ClientName      string        `json:"client_name"`
	PoolSize        int           `json:"pool_size"`
	MinIdleConns    int           `json:"min_idle_conns"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
	DialTimeout     time.Duration `json:"dial_timeout"`
}

// PriorityConfig holds priority selection configuration
type PriorityConfig struct {
	TieBreaker string // "round_robin" or "random"
}

// Load creates a new Config with values from environment variables or defaults
func Load() *Config {
	cfg := &Config{
		// DNS Server defaults
		DNSPort:              "5353",
		MaxConcurrentQueries: 1000,
		ShutdownTimeout:      30 * time.Second,
		LogLevel:             "info",

		// Database defaults
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "dnsuser",
			Password:        "dnspass",
			DBName:          "dnsdb",
			SSLMode:         "disable",
			ConnectionName:  "dns_primary",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 2 * time.Minute,
		},

		// Cache defaults
		Cache: CacheConfig{
			Enabled:         true,
			MaxEntries:      10000,
			CleanupInterval: 60 * time.Second,
			DefaultTTL:      300 * time.Second,
		},

		// Redis defaults
		Redis: RedisConfig{
			Enabled:         false, // Disabled by default
			Address:         "localhost:6379",
			Password:        "",
			Database:        0,
			ClientName:      "errantdns",
			PoolSize:        10,
			MinIdleConns:    3,
			ConnMaxIdleTime: 240 * time.Second,
			DialTimeout:     2 * time.Second,
		},

		// Priority defaults
		Priority: PriorityConfig{
			TieBreaker: "round_robin",
		},

		// Logging defaults
		Logging: LoggingConfig{
			Level:           "INFO",
			Directory:       "logs",
			AppLogFile:      "app.log",
			QueryLogFile:    "queries.log",
			ErrorLogFile:    "errors.log",
			EnableConsole:   true,
			QuerySampleRate: 0.01, // 1%
			BufferSize:      1000,
		},
	}

	// Override with environment variables
	loadDNSConfig(cfg)
	loadDatabaseConfig(cfg)
	loadCacheConfig(cfg)
	loadRedisConfig(cfg)
	loadPriorityConfig(cfg)
	loadLoggingConfig(cfg)
	loadServerConfig(cfg)

	return cfg
}

func loadLoggingConfig(cfg *Config) {
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		cfg.Logging.Level = strings.ToUpper(env)
	}

	if env := os.Getenv("LOG_DIRECTORY"); env != "" {
		cfg.Logging.Directory = env
	}

	if env := os.Getenv("LOG_APP_FILE"); env != "" {
		cfg.Logging.AppLogFile = env
	}

	if env := os.Getenv("LOG_QUERY_FILE"); env != "" {
		cfg.Logging.QueryLogFile = env
	}

	if env := os.Getenv("LOG_ERROR_FILE"); env != "" {
		cfg.Logging.ErrorLogFile = env
	}

	if env := os.Getenv("LOG_ENABLE_CONSOLE"); env != "" {
		if val, err := strconv.ParseBool(env); err == nil {
			cfg.Logging.EnableConsole = val
		}
	}

	if env := os.Getenv("LOG_QUERY_SAMPLE_RATE"); env != "" {
		if val, err := strconv.ParseFloat(env, 64); err == nil && val >= 0 && val <= 1 {
			cfg.Logging.QuerySampleRate = val
		}
	}

	if env := os.Getenv("LOG_BUFFER_SIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val > 0 {
			cfg.Logging.BufferSize = val
		}
	}
}

// loadDNSConfig loads DNS-specific configuration from environment
func loadDNSConfig(cfg *Config) {
	if env := os.Getenv("DNS_PORT"); env != "" {
		cfg.DNSPort = env
	}
}

// loadDatabaseConfig loads database configuration from environment
func loadDatabaseConfig(cfg *Config) {
	if env := os.Getenv("DATABASE_URL"); env != "" {
		// If DATABASE_URL is provided, it takes precedence
		// Format: postgres://user:password@host:port/dbname?sslmode=disable
		// For now, we'll keep individual settings approach
		// TODO: Add URL parsing if needed
	}

	if env := os.Getenv("DB_HOST"); env != "" {
		cfg.Database.Host = env
	}

	if env := os.Getenv("DB_PORT"); env != "" {
		if port, err := strconv.Atoi(env); err == nil && port > 0 {
			cfg.Database.Port = port
		}
	}

	if env := os.Getenv("DB_USER"); env != "" {
		cfg.Database.User = env
	}

	if env := os.Getenv("DB_PASSWORD"); env != "" {
		cfg.Database.Password = env
	}

	if env := os.Getenv("DB_NAME"); env != "" {
		cfg.Database.DBName = env
	}

	if env := os.Getenv("DB_SSL_MODE"); env != "" {
		cfg.Database.SSLMode = env
	}

	if env := os.Getenv("DB_CONNECTION_NAME"); env != "" {
		cfg.Database.ConnectionName = env
	}

	if env := os.Getenv("DB_MAX_OPEN_CONNS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val > 0 {
			cfg.Database.MaxOpenConns = val
		}
	}

	if env := os.Getenv("DB_MAX_IDLE_CONNS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val >= 0 {
			cfg.Database.MaxIdleConns = val
		}
	}

	if env := os.Getenv("DB_CONN_MAX_LIFETIME"); env != "" {
		if val, err := time.ParseDuration(env); err == nil {
			cfg.Database.ConnMaxLifetime = val
		}
	}

	if env := os.Getenv("DB_CONN_MAX_IDLE_TIME"); env != "" {
		if val, err := time.ParseDuration(env); err == nil {
			cfg.Database.ConnMaxIdleTime = val
		}
	}
}

// loadCacheConfig loads cache configuration from environment
func loadCacheConfig(cfg *Config) {
	if env := os.Getenv("CACHE_ENABLED"); env != "" {
		if val, err := strconv.ParseBool(env); err == nil {
			cfg.Cache.Enabled = val
		}
	}

	if env := os.Getenv("CACHE_MAX_ENTRIES"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val > 0 {
			cfg.Cache.MaxEntries = val
		}
	}

	if env := os.Getenv("CACHE_CLEANUP_INTERVAL"); env != "" {
		if val, err := time.ParseDuration(env); err == nil {
			cfg.Cache.CleanupInterval = val
		}
	}

	if env := os.Getenv("CACHE_DEFAULT_TTL"); env != "" {
		if val, err := time.ParseDuration(env); err == nil {
			cfg.Cache.DefaultTTL = val
		}
	}
}

// loadRedisConfig loads Redis configuration from environment
func loadRedisConfig(cfg *Config) {
	if env := os.Getenv("REDIS_ENABLED"); env != "" {
		if val, err := strconv.ParseBool(env); err == nil {
			cfg.Redis.Enabled = val
		}
	}

	if env := os.Getenv("REDIS_ADDRESS"); env != "" {
		cfg.Redis.Address = env
	}

	if env := os.Getenv("REDIS_PASSWORD"); env != "" {
		cfg.Redis.Password = env
	}

	if env := os.Getenv("REDIS_DATABASE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val >= 0 {
			cfg.Redis.Database = val
		}
	}

	if env := os.Getenv("REDIS_CLIENT_NAME"); env != "" {
		cfg.Redis.ClientName = env
	}

	if env := os.Getenv("REDIS_POOL_SIZE"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val > 0 {
			cfg.Redis.PoolSize = val
		}
	}

	if env := os.Getenv("REDIS_MIN_IDLE_CONNS"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val >= 0 {
			cfg.Redis.MinIdleConns = val
		}
	}

	if env := os.Getenv("REDIS_CONN_MAX_IDLE_TIME"); env != "" {
		if val, err := time.ParseDuration(env); err == nil {
			cfg.Redis.ConnMaxIdleTime = val
		}
	}

	if env := os.Getenv("REDIS_DIAL_TIMEOUT"); env != "" {
		if val, err := time.ParseDuration(env); err == nil {
			cfg.Redis.DialTimeout = val
		}
	}
}

// loadPriorityConfig loads priority configuration from environment
func loadPriorityConfig(cfg *Config) {
	if env := os.Getenv("PRIORITY_TIE_BREAKER"); env != "" {
		if env == "round_robin" || env == "random" {
			cfg.Priority.TieBreaker = env
		}
	}
}

// loadServerConfig loads server behavior configuration from environment
func loadServerConfig(cfg *Config) {
	if env := os.Getenv("MAX_CONCURRENT_QUERIES"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val > 0 {
			cfg.MaxConcurrentQueries = val
		}
	}

	if env := os.Getenv("SHUTDOWN_TIMEOUT"); env != "" {
		if val, err := time.ParseDuration(env); err == nil {
			cfg.ShutdownTimeout = val
		}
	}

	if env := os.Getenv("LOG_LEVEL"); env != "" {
		cfg.LogLevel = env
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// DNS validation
	if c.DNSPort == "" {
		return &ValidationError{Field: "DNSPort", Message: "cannot be empty"}
	}

	// Database validation
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config error: %w", err)
	}

	// Cache validation
	if err := c.Cache.Validate(); err != nil {
		return fmt.Errorf("cache config error: %w", err)
	}

	// Redis validation
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("redis config error: %w", err)
	}

	// Priority validation
	if err := c.Priority.Validate(); err != nil {
		return fmt.Errorf("priority config error: %w", err)
	}

	// Server validation
	if c.MaxConcurrentQueries <= 0 {
		return &ValidationError{Field: "MaxConcurrentQueries", Message: "must be greater than 0"}
	}

	// Logging validation
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config error: %w", err)
	}

	return nil
}

// Validate validates database configuration
func (db *DatabaseConfig) Validate() error {
	if db.Host == "" {
		return &ValidationError{Field: "Host", Message: "cannot be empty"}
	}

	if db.Port <= 0 || db.Port > 65535 {
		return &ValidationError{Field: "Port", Message: "must be between 1 and 65535"}
	}

	if db.User == "" {
		return &ValidationError{Field: "User", Message: "cannot be empty"}
	}

	if db.DBName == "" {
		return &ValidationError{Field: "DBName", Message: "cannot be empty"}
	}

	if db.ConnectionName == "" {
		return &ValidationError{Field: "ConnectionName", Message: "cannot be empty"}
	}

	if db.MaxOpenConns <= 0 {
		return &ValidationError{Field: "MaxOpenConns", Message: "must be greater than 0"}
	}

	if db.MaxIdleConns < 0 {
		return &ValidationError{Field: "MaxIdleConns", Message: "cannot be negative"}
	}

	return nil
}

// Validate validates cache configuration
func (cache *CacheConfig) Validate() error {
	if cache.Enabled {
		if cache.MaxEntries <= 0 {
			return &ValidationError{Field: "MaxEntries", Message: "must be greater than 0 when cache is enabled"}
		}

		if cache.CleanupInterval < 0 {
			return &ValidationError{Field: "CleanupInterval", Message: "cannot be negative"}
		}

		if cache.DefaultTTL < 0 {
			return &ValidationError{Field: "DefaultTTL", Message: "cannot be negative"}
		}
	}

	return nil
}

func (logging *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	}

	if !validLevels[strings.ToUpper(logging.Level)] {
		return &ValidationError{Field: "Level", Message: "must be DEBUG, INFO, WARN, or ERROR"}
	}

	if logging.Directory == "" {
		return &ValidationError{Field: "Directory", Message: "cannot be empty"}
	}

	if logging.QuerySampleRate < 0 || logging.QuerySampleRate > 1 {
		return &ValidationError{Field: "QuerySampleRate", Message: "must be between 0 and 1"}
	}

	if logging.BufferSize <= 0 {
		return &ValidationError{Field: "BufferSize", Message: "must be greater than 0"}
	}

	return nil
}

// Validate validates Redis configuration
func (redis *RedisConfig) Validate() error {
	if !redis.Enabled {
		return nil // Skip validation if Redis is disabled
	}

	if redis.Address == "" {
		return &ValidationError{Field: "Redis.Address", Message: "cannot be empty when Redis is enabled"}
	}

	if redis.ClientName == "" {
		return &ValidationError{Field: "Redis.ClientName", Message: "cannot be empty when Redis is enabled"}
	}

	if redis.Database < 0 {
		return &ValidationError{Field: "Redis.Database", Message: "cannot be negative"}
	}

	if redis.PoolSize <= 0 {
		return &ValidationError{Field: "Redis.PoolSize", Message: "must be greater than 0"}
	}

	if redis.MinIdleConns < 0 {
		return &ValidationError{Field: "Redis.MinIdleConns", Message: "cannot be negative"}
	}

	if redis.MinIdleConns > redis.PoolSize {
		return &ValidationError{Field: "Redis.MinIdleConns", Message: "cannot be greater than pool size"}
	}

	return nil
}

// Validate validates priority configuration
func (priority *PriorityConfig) Validate() error {
	if priority.TieBreaker != "round_robin" && priority.TieBreaker != "random" {
		return &ValidationError{Field: "TieBreaker", Message: "must be 'round_robin' or 'random'"}
	}

	return nil
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation error: %s %s", e.Field, e.Message)
}
