// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"
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

	// Priority configuration
	Priority PriorityConfig

	// Server behavior
	MaxConcurrentQueries int
	ShutdownTimeout      time.Duration

	// Logging
	LogLevel string
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

		// Priority defaults
		Priority: PriorityConfig{
			TieBreaker: "round_robin",
		},
	}

	// Override with environment variables
	loadDNSConfig(cfg)
	loadDatabaseConfig(cfg)
	loadCacheConfig(cfg)
	loadPriorityConfig(cfg)
	loadServerConfig(cfg)

	return cfg
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

	// Priority validation
	if err := c.Priority.Validate(); err != nil {
		return fmt.Errorf("priority config error: %w", err)
	}

	// Server validation
	if c.MaxConcurrentQueries <= 0 {
		return &ValidationError{Field: "MaxConcurrentQueries", Message: "must be greater than 0"}
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
