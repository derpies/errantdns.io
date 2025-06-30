// internal/logging/logger.go
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents logging levels
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// Config holds logging configuration
type Config struct {
	Level           LogLevel `json:"level"`
	Directory       string   `json:"directory"`
	AppLogFile      string   `json:"app_log_file"`
	QueryLogFile    string   `json:"query_log_file"`
	ErrorLogFile    string   `json:"error_log_file"`
	EnableConsole   bool     `json:"enable_console"`
	QuerySampleRate float64  `json:"query_sample_rate"`
	BufferSize      int      `json:"buffer_size"`
}

// DefaultConfig returns default logging configuration
func DefaultConfig() *Config {
	return &Config{
		Level:           LevelInfo,
		Directory:       "logs",
		AppLogFile:      "app.log",
		QueryLogFile:    "queries.log",
		ErrorLogFile:    "errors.log",
		EnableConsole:   true,
		QuerySampleRate: 0.01, // 1%
		BufferSize:      1000,
	}
}

// Logger represents the global logger instance
type Logger struct {
	config      *Config
	appLogger   *slog.Logger
	queryLogger *slog.Logger
	errorLogger *slog.Logger

	// Query sampling
	sampleRNG   *rand.Rand
	sampleMutex sync.Mutex

	// Performance counters
	queriesLogged  int64
	queriesSampled int64
	errorsLogged   int64

	// File handles for cleanup
	appFile   *os.File
	queryFile *os.File
	errorFile *os.File
}

var (
	globalLogger *Logger
	once         sync.Once
)

// Initialize sets up the global logger
func Initialize(config *Config) error {
	var err error
	once.Do(func() {
		globalLogger, err = newLogger(config)
	})
	return err
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		// Fallback to default config if not initialized
		_ = Initialize(DefaultConfig())
	}
	return globalLogger
}

// newLogger creates a new logger instance
func newLogger(config *Config) (*Logger, error) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(config.Directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logger := &Logger{
		config:    config,
		sampleRNG: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Set up application logger
	if err := logger.setupAppLogger(); err != nil {
		return nil, fmt.Errorf("failed to setup app logger: %w", err)
	}

	// Set up query logger
	if err := logger.setupQueryLogger(); err != nil {
		return nil, fmt.Errorf("failed to setup query logger: %w", err)
	}

	// Set up error logger
	if err := logger.setupErrorLogger(); err != nil {
		return nil, fmt.Errorf("failed to setup error logger: %w", err)
	}

	return logger, nil
}

// setupAppLogger configures the application logger
func (l *Logger) setupAppLogger() error {
	writers := []io.Writer{}

	// File output
	appPath := filepath.Join(l.config.Directory, l.config.AppLogFile)
	appFile, err := os.OpenFile(appPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open app log file: %w", err)
	}
	l.appFile = appFile
	writers = append(writers, appFile)

	// Console output
	if l.config.EnableConsole {
		writers = append(writers, os.Stdout)
	}

	multiWriter := io.MultiWriter(writers...)

	opts := &slog.HandlerOptions{
		Level: l.getSlogLevel(),
	}

	handler := slog.NewJSONHandler(multiWriter, opts)
	l.appLogger = slog.New(handler)

	return nil
}

// setupQueryLogger configures the query logger
func (l *Logger) setupQueryLogger() error {
	queryPath := filepath.Join(l.config.Directory, l.config.QueryLogFile)
	queryFile, err := os.OpenFile(queryPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open query log file: %w", err)
	}
	l.queryFile = queryFile

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Query logger accepts all levels
	}

	handler := slog.NewJSONHandler(queryFile, opts)
	l.queryLogger = slog.New(handler)

	return nil
}

// setupErrorLogger configures the error logger
func (l *Logger) setupErrorLogger() error {
	errorPath := filepath.Join(l.config.Directory, l.config.ErrorLogFile)
	errorFile, err := os.OpenFile(errorPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open error log file: %w", err)
	}
	l.errorFile = errorFile

	opts := &slog.HandlerOptions{
		Level: slog.LevelWarn, // Errors and warnings only
	}

	handler := slog.NewJSONHandler(errorFile, opts)
	l.errorLogger = slog.New(handler)

	return nil
}

// getSlogLevel converts our LogLevel to slog.Level
func (l *Logger) getSlogLevel() slog.Level {
	switch l.config.Level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// shouldSampleQuery determines if a query should be logged based on sampling rate
func (l *Logger) shouldSampleQuery() bool {
	if l.config.Level == LevelDebug {
		return true // Always log in debug mode
	}

	l.sampleMutex.Lock()
	defer l.sampleMutex.Unlock()

	return l.sampleRNG.Float64() < l.config.QuerySampleRate
}

// Application Logging Methods

// Info logs an informational message
func (l *Logger) Info(component, message string, fields ...interface{}) {
	l.appLogger.Info(message, append([]interface{}{"component", component}, fields...)...)
}

// Warn logs a warning message
func (l *Logger) Warn(component, message string, fields ...interface{}) {
	l.appLogger.Warn(message, append([]interface{}{"component", component}, fields...)...)
}

// Error logs an error message
func (l *Logger) Error(component, message string, err error, fields ...interface{}) {
	allFields := append([]interface{}{"component", component}, fields...)
	if err != nil {
		allFields = append(allFields, "error", err.Error())
	}
	l.appLogger.Error(message, allFields...)
}

// Debug logs a debug message
func (l *Logger) Debug(component, message string, fields ...interface{}) {
	l.appLogger.Debug(message, append([]interface{}{"component", component}, fields...)...)
}

// Query Logging Methods

// LogQuery logs a DNS query with sampling
func (l *Logger) LogQuery(domain, queryType, result, source string, responseTime time.Duration) {
	if !l.shouldSampleQuery() {
		return
	}

	l.queryLogger.Info("dns_query",
		"domain", domain,
		"type", queryType,
		"result", result,
		"source", source,
		"response_time_ms", responseTime.Milliseconds(),
		"timestamp", time.Now().Unix(),
	)

	l.queriesLogged++
}

// LogQueryDebug logs a DNS query with full debug information
func (l *Logger) LogQueryDebug(domain, queryType, result, source string, responseTime time.Duration, extra map[string]interface{}) {
	if l.config.Level != LevelDebug {
		return
	}

	fields := []interface{}{
		"domain", domain,
		"type", queryType,
		"result", result,
		"source", source,
		"response_time_ms", responseTime.Milliseconds(),
		"timestamp", time.Now().Unix(),
	}

	// Add extra debug fields
	for k, v := range extra {
		fields = append(fields, k, v)
	}

	l.queryLogger.Debug("dns_query_debug", fields...)
}

// Error Event Logging Methods

// LogNXDOMAIN logs NXDOMAIN responses
func (l *Logger) LogNXDOMAIN(domain, queryType string, responseTime time.Duration) {
	l.errorLogger.Warn("nxdomain",
		"event_type", "nxdomain",
		"domain", domain,
		"type", queryType,
		"response_time_ms", responseTime.Milliseconds(),
		"timestamp", time.Now().Unix(),
	)
	l.errorsLogged++
}

// LogQueryTimeout logs query timeouts
func (l *Logger) LogQueryTimeout(domain, queryType string, timeout time.Duration) {
	l.errorLogger.Error("query_timeout",
		"event_type", "timeout",
		"domain", domain,
		"type", queryType,
		"timeout_ms", timeout.Milliseconds(),
		"timestamp", time.Now().Unix(),
	)
	l.errorsLogged++
}

// LogCacheMiss logs cache misses for analysis
func (l *Logger) LogCacheMiss(domain, queryType string, cacheLevel string) {
	l.errorLogger.Info("cache_miss",
		"event_type", "cache_miss",
		"domain", domain,
		"type", queryType,
		"cache_level", cacheLevel,
		"timestamp", time.Now().Unix(),
	)
}

// LogMalformedQuery logs malformed DNS queries
func (l *Logger) LogMalformedQuery(rawQuery string, error string) {
	l.errorLogger.Warn("malformed_query",
		"event_type", "malformed_query",
		"raw_query", rawQuery,
		"error", error,
		"timestamp", time.Now().Unix(),
	)
	l.errorsLogged++
}

// GetStats returns logging statistics
func (l *Logger) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"queries_logged":  l.queriesLogged,
		"queries_sampled": l.queriesSampled,
		"errors_logged":   l.errorsLogged,
		"sample_rate":     l.config.QuerySampleRate,
		"log_level":       string(l.config.Level),
	}
}

// Close closes all log files
func (l *Logger) Close() error {
	var lastErr error

	if l.appFile != nil {
		if err := l.appFile.Close(); err != nil {
			lastErr = err
		}
	}

	if l.queryFile != nil {
		if err := l.queryFile.Close(); err != nil {
			lastErr = err
		}
	}

	if l.errorFile != nil {
		if err := l.errorFile.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Global convenience functions for easy migration

// Info logs an informational message using the global logger
func Info(component, message string, fields ...interface{}) {
	GetLogger().Info(component, message, fields...)
}

// Warn logs a warning message using the global logger
func Warn(component, message string, fields ...interface{}) {
	GetLogger().Warn(component, message, fields...)
}

// Error logs an error message using the global logger
func Error(component, message string, err error, fields ...interface{}) {
	GetLogger().Error(component, message, err, fields...)
}

// Debug logs a debug message using the global logger
func Debug(component, message string, fields ...interface{}) {
	GetLogger().Debug(component, message, fields...)
}

// LogQuery logs a DNS query using the global logger
func LogQuery(domain, queryType, result, source string, responseTime time.Duration) {
	GetLogger().LogQuery(domain, queryType, result, source, responseTime)
}

// LogNXDOMAIN logs NXDOMAIN responses using the global logger
func LogNXDOMAIN(domain, queryType string, responseTime time.Duration) {
	GetLogger().LogNXDOMAIN(domain, queryType, responseTime)
}

// LogQueryTimeout logs query timeouts using the global logger
func LogQueryTimeout(domain, queryType string, timeout time.Duration) {
	GetLogger().LogQueryTimeout(domain, queryType, timeout)
}

// LogCacheMiss logs cache misses using the global logger
func LogCacheMiss(domain, queryType string, cacheLevel string) {
	GetLogger().LogCacheMiss(domain, queryType, cacheLevel)
}

// LogMalformedQuery logs malformed queries using the global logger
func LogMalformedQuery(rawQuery string, error string) {
	GetLogger().LogMalformedQuery(rawQuery, error)
}
