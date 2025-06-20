// internal/storage/postgres.go
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"errantdns.io/internal/models"
	"errantdns.io/internal/pgsqlpool"
)

// Storage interface defines the contract for DNS record storage
type Storage interface {
	// Query operations
	LookupRecord(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error)
	LookupRecords(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error)

	// Management operations
	CreateRecord(ctx context.Context, record *models.DNSRecord) error
	UpdateRecord(ctx context.Context, record *models.DNSRecord) error
	DeleteRecord(ctx context.Context, id int) error
	DeleteRecords(ctx context.Context, name string, recordType string) error

	// System operations
	Health(ctx context.Context) error
	Close() error
}

// PostgresStorage implements Storage interface using the improved pgsqlpool
type PostgresStorage struct {
	pool           *pgsqlpool.Pool
	connectionName string
}

// Config holds configuration for PostgreSQL storage
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Host:            "localhost",
		Port:            5432,
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(ctx context.Context, pool *pgsqlpool.Pool, connectionName string, config *Config) (*PostgresStorage, error) {
	// Create connection config
	connConfig := &pgsqlpool.ConnectionConfig{
		Host:            config.Host,
		Port:            config.Port,
		User:            config.User,
		Password:        config.Password,
		DBName:          config.DBName,
		SSLMode:         config.SSLMode,
		MaxOpenConns:    config.MaxOpenConns,
		MaxIdleConns:    config.MaxIdleConns,
		ConnMaxLifetime: config.ConnMaxLifetime,
		ConnMaxIdleTime: config.ConnMaxIdleTime,
	}

	// Add the connection to the provided pool
	if err := pool.AddConnection(ctx, connectionName, connConfig); err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	return &PostgresStorage{
		pool:           pool,
		connectionName: connectionName,
	}, nil
}

// LookupRecord finds a single DNS record matching the query
// Returns the highest priority record if multiple records exist
func (s *PostgresStorage) LookupRecord(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error) {
	sqlQuery := `
		SELECT id, name, record_type, target, ttl, priority, created_at, updated_at
		FROM dns_records 
		WHERE LOWER(name) = LOWER($1) AND record_type = $2
		ORDER BY priority DESC
		LIMIT 1
	`

	row := s.pool.QueryRow(ctx, s.connectionName, sqlQuery, query.Name, query.Type.String())

	var record models.DNSRecord
	err := row.Scan(
		&record.ID,
		&record.Name,
		&record.RecordType,
		&record.Target,
		&record.TTL,
		&record.Priority,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No record found, not an error
		}
		return nil, fmt.Errorf("failed to scan record for %s %s: %w", query.Name, query.Type, err)
	}

	return &record, nil
}

// LookupRecords finds all DNS records matching the query, ordered by priority
func (s *PostgresStorage) LookupRecords(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	sqlQuery := `
		SELECT id, name, record_type, target, ttl, priority, created_at, updated_at
		FROM dns_records 
		WHERE LOWER(name) = LOWER($1) AND record_type = $2
		ORDER BY priority DESC
	`

	rows, err := s.pool.Query(ctx, s.connectionName, sqlQuery, query.Name, query.Type.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query records for %s %s: %w", query.Name, query.Type, err)
	}
	defer rows.Close()

	var records []*models.DNSRecord
	for rows.Next() {
		var record models.DNSRecord
		err := rows.Scan(
			&record.ID,
			&record.Name,
			&record.RecordType,
			&record.Target,
			&record.TTL,
			&record.Priority,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating records: %w", err)
	}

	return records, nil
}

// CreateRecord inserts a new DNS record
func (s *PostgresStorage) CreateRecord(ctx context.Context, record *models.DNSRecord) error {
	// Validate and normalize the record
	if err := record.Validate(); err != nil {
		return fmt.Errorf("invalid record: %w", err)
	}
	record.Normalize()

	sqlQuery := `
		INSERT INTO dns_records (name, record_type, target, ttl, priority)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	row := s.pool.QueryRow(ctx, s.connectionName, sqlQuery,
		record.Name,
		record.RecordType,
		record.Target,
		record.TTL,
		record.Priority,
	)

	err := row.Scan(&record.ID, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create record %s %s: %w", record.Name, record.RecordType, err)
	}

	return nil
}

// UpdateRecord updates an existing DNS record
func (s *PostgresStorage) UpdateRecord(ctx context.Context, record *models.DNSRecord) error {
	// Validate and normalize the record
	if err := record.Validate(); err != nil {
		return fmt.Errorf("invalid record: %w", err)
	}
	record.Normalize()

	sqlQuery := `
		UPDATE dns_records 
		SET name = $1, record_type = $2, target = $3, ttl = $4, priority = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING updated_at
	`

	row := s.pool.QueryRow(ctx, s.connectionName, sqlQuery,
		record.Name,
		record.RecordType,
		record.Target,
		record.TTL,
		record.Priority,
		record.ID,
	)

	err := row.Scan(&record.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("record with ID %d not found", record.ID)
		}
		return fmt.Errorf("failed to update record ID %d: %w", record.ID, err)
	}

	return nil
}

// DeleteRecord deletes a DNS record by ID
func (s *PostgresStorage) DeleteRecord(ctx context.Context, id int) error {
	sqlQuery := `DELETE FROM dns_records WHERE id = $1`

	result, err := s.pool.Exec(ctx, s.connectionName, sqlQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete record ID %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("record with ID %d not found", id)
	}

	return nil
}

// DeleteRecords deletes all DNS records matching name and optionally type
func (s *PostgresStorage) DeleteRecords(ctx context.Context, name string, recordType string) error {
	normalizedName := models.NormalizeDomainName(name)

	var sqlQuery string
	var args []interface{}

	if recordType == "" {
		// Delete all records for the domain
		sqlQuery = `DELETE FROM dns_records WHERE LOWER(name) = LOWER($1)`
		args = []interface{}{normalizedName}
	} else {
		// Delete specific record type for the domain
		sqlQuery = `DELETE FROM dns_records WHERE LOWER(name) = LOWER($1) AND record_type = $2`
		args = []interface{}{normalizedName, recordType}
	}

	result, err := s.pool.Exec(ctx, s.connectionName, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to delete records for %s %s: %w", name, recordType, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no records found for %s %s", name, recordType)
	}

	return nil
}

// Health checks if the database connection is healthy
func (s *PostgresStorage) Health(ctx context.Context) error {
	return s.pool.HealthCheck(ctx, s.connectionName)
}

// Close closes the database connection pool
func (s *PostgresStorage) Close() error {
	return s.pool.Close()
}

// InitializeSchema creates the DNS records table using a schema file
func (s *PostgresStorage) InitializeSchema(ctx context.Context, schemaFilePath string) error {
	return s.pool.ExecSchemaFile(ctx, s.connectionName, schemaFilePath)
}
