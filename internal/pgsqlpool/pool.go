// internal/pgsqlpool/pool.go
package pgsqlpool

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// ConnectionConfig holds configuration for a database connection
type ConnectionConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string // disable, require, verify-ca, verify-full

	// Pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConnectionConfig returns a config with sensible defaults
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Host:            "localhost",
		Port:            5432,
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}
}

// DSN returns the PostgreSQL data source name for this config
func (c *ConnectionConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// Validate checks if the connection config is valid
func (c *ConnectionConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.User == "" {
		return fmt.Errorf("user cannot be empty")
	}
	if c.DBName == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("max open connections must be greater than 0")
	}
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("max idle connections cannot be negative")
	}
	return nil
}

// Pool manages named database connections
type Pool struct {
	mu          sync.RWMutex
	connections map[string]*sql.DB
}

// NewPool creates a new connection pool
func NewPool() *Pool {
	return &Pool{
		connections: make(map[string]*sql.DB),
	}
}

// AddConnection creates and adds a new named database connection
func (p *Pool) AddConnection(ctx context.Context, name string, config *ConnectionConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config for connection %s: %w", name, err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if connection already exists
	if _, exists := p.connections[name]; exists {
		return fmt.Errorf("connection %s already exists", name)
	}

	// Create the connection
	db, err := sql.Open("postgres", config.DSN())
	if err != nil {
		return fmt.Errorf("failed to open connection %s: %w", name, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping connection %s: %w", name, err)
	}

	// Store the connection
	p.connections[name] = db

	return nil
}

// GetConnection returns a named database connection
func (p *Pool) GetConnection(name string) (*sql.DB, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	db, exists := p.connections[name]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", name)
	}

	return db, nil
}

// RemoveConnection closes and removes a named connection
func (p *Pool) RemoveConnection(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	db, exists := p.connections[name]
	if !exists {
		return fmt.Errorf("connection %s not found", name)
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close connection %s: %w", name, err)
	}

	delete(p.connections, name)
	return nil
}

// ConnectionExists checks if a named connection exists
func (p *Pool) ConnectionExists(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, exists := p.connections[name]
	return exists
}

// ListConnections returns a list of all connection names
func (p *Pool) ListConnections() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.connections))
	for name := range p.connections {
		names = append(names, name)
	}
	return names
}

// HealthCheck checks if a named connection is healthy
func (p *Pool) HealthCheck(ctx context.Context, name string) error {
	db, err := p.GetConnection(name)
	if err != nil {
		return err
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("health check failed for connection %s: %w", name, err)
	}

	return nil
}

// HealthCheckAll checks all connections and returns any errors
func (p *Pool) HealthCheckAll(ctx context.Context) map[string]error {
	p.mu.RLock()
	names := make([]string, 0, len(p.connections))
	for name := range p.connections {
		names = append(names, name)
	}
	p.mu.RUnlock()

	errors := make(map[string]error)
	for _, name := range names {
		if err := p.HealthCheck(ctx, name); err != nil {
			errors[name] = err
		}
	}

	return errors
}

// Close closes all connections
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for name, db := range p.connections {
		if err := db.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close connection %s: %w", name, err)
		}
	}

	// Clear the map
	p.connections = make(map[string]*sql.DB)

	return lastErr
}

// Query executes a query on a named connection
func (p *Pool) Query(ctx context.Context, connectionName, query string, args ...interface{}) (*sql.Rows, error) {
	db, err := p.GetConnection(connectionName)
	if err != nil {
		return nil, err
	}

	return db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row on a named connection
func (p *Pool) QueryRow(ctx context.Context, connectionName, query string, args ...interface{}) *sql.Row {
	db, err := p.GetConnection(connectionName)
	if err != nil {
		// Return a row that will error when scanned
		return &sql.Row{}
	}

	return db.QueryRowContext(ctx, query, args...)
}

// Exec executes a statement on a named connection
func (p *Pool) Exec(ctx context.Context, connectionName, query string, args ...interface{}) (sql.Result, error) {
	db, err := p.GetConnection(connectionName)
	if err != nil {
		return nil, err
	}

	return db.ExecContext(ctx, query, args...)
}

// Transaction executes a function within a transaction on a named connection
func (p *Pool) Transaction(ctx context.Context, connectionName string, fn func(*sql.Tx) error) error {
	db, err := p.GetConnection(connectionName)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback failed: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExecSchemaFile executes SQL statements from a file on a named connection
func (p *Pool) ExecSchemaFile(ctx context.Context, connectionName, filePath string) error {
	db, err := p.GetConnection(connectionName)
	if err != nil {
		return err
	}

	// Read the SQL file
	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", filePath, err)
	}

	// Execute the SQL statements
	_, err = db.ExecContext(ctx, string(sqlBytes))
	if err != nil {
		return fmt.Errorf("failed to execute schema from %s: %w", filePath, err)
	}

	return nil
}
