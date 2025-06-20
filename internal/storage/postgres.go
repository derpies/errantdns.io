// internal/storage/postgres.go
package storage

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"time"

	"errantdns.io/internal/models"
	"errantdns.io/internal/pgsqlpool"
)

// Storage interface defines the contract for DNS record storage
type Storage interface {
	// Query operations
	LookupRecord(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error)
	LookupRecords(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error)
	LookupRecordGroup(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error)

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
	tieBreaker     string
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
func NewPostgresStorage(ctx context.Context, pool *pgsqlpool.Pool, connectionName string, config *Config, tieBreaker string) (*PostgresStorage, error) {
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
		tieBreaker:     tieBreaker,
	}, nil
}

// LookupRecord finds a single DNS record matching the query using priority selection
// Returns one record from the lowest priority group with tie-breaking
func (s *PostgresStorage) LookupRecord(ctx context.Context, query *models.LookupQuery) (*models.DNSRecord, error) {
	// Get all records in the highest priority group (lowest priority number)
	records, err := s.LookupRecordGroup(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil // No records found
	}

	// If only one record, return it
	if len(records) == 1 {
		return records[0], nil
	}

	// Apply tie-breaking for multiple records
	selected := s.selectFromGroup(records, query)
	return selected, nil
}

// LookupRecords finds all DNS records matching the query, ordered by priority
func (s *PostgresStorage) LookupRecords(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	sqlQuery := `
		SELECT id, name, record_type, target, ttl, priority, created_at, updated_at
		FROM dns_records 
		WHERE LOWER(name) = LOWER($1) AND record_type = $2
		ORDER BY priority ASC
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

// LookupRecordGroup finds all records with the same lowest priority for the query
func (s *PostgresStorage) LookupRecordGroup(ctx context.Context, query *models.LookupQuery) ([]*models.DNSRecord, error) {
	// First, get the lowest priority value
	minPriorityQuery := `
		SELECT MIN(priority) 
		FROM dns_records 
		WHERE LOWER(name) = LOWER($1) AND record_type = $2
	`

	row := s.pool.QueryRow(ctx, s.connectionName, minPriorityQuery, query.Name, query.Type.String())

	var minPriority sql.NullInt32
	err := row.Scan(&minPriority)
	if err != nil {
		if err == sql.ErrNoRows || !minPriority.Valid {
			return nil, nil // No records found
		}
		return nil, fmt.Errorf("failed to get min priority for %s %s: %w", query.Name, query.Type, err)
	}

	// Now get all records with that minimum priority
	recordsQuery := `
		SELECT id, name, record_type, target, ttl, priority, created_at, updated_at
		FROM dns_records 
		WHERE LOWER(name) = LOWER($1) AND record_type = $2 AND priority = $3
		ORDER BY id ASC
	`

	rows, err := s.pool.Query(ctx, s.connectionName, recordsQuery, query.Name, query.Type.String(), minPriority.Int32)
	if err != nil {
		return nil, fmt.Errorf("failed to query record group for %s %s: %w", query.Name, query.Type, err)
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
		return nil, fmt.Errorf("error iterating record group: %w", err)
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

// selectFromGroup applies tie-breaking logic to select one record from a group
func (s *PostgresStorage) selectFromGroup(records []*models.DNSRecord, query *models.LookupQuery) *models.DNSRecord {
	if len(records) == 0 {
		return nil
	}

	if len(records) == 1 {
		return records[0]
	}

	switch s.tieBreaker {
	case "random":
		// Use query-based seed for consistency within same query
		seed := s.generateSeed(query)
		rng := rand.New(rand.NewSource(seed))
		index := rng.Intn(len(records))
		return records[index]

	case "round_robin":
		fallthrough
	default:
		// Round-robin based on time and query hash
		index := s.roundRobinIndex(query, len(records))
		return records[index]
	}
}

// generateSeed creates a deterministic seed based on the query
func (s *PostgresStorage) generateSeed(query *models.LookupQuery) int64 {
	h := fnv.New64a()
	h.Write([]byte(query.Name))
	h.Write([]byte(query.Type.String()))
	// Add some time component for variation
	timeComponent := time.Now().Unix() / 300 // Changes every 5 minutes
	h.Write([]byte(fmt.Sprintf("%d", timeComponent)))
	return int64(h.Sum64())
}

// roundRobinIndex calculates round-robin index based on time and query
func (s *PostgresStorage) roundRobinIndex(query *models.LookupQuery, count int) int {
	if count <= 1 {
		return 0
	}

	// Create deterministic hash of query
	h := md5.New()
	h.Write([]byte(query.Name))
	h.Write([]byte(query.Type.String()))
	queryHash := h.Sum(nil)

	// Convert first 8 bytes to uint64
	queryValue := binary.BigEndian.Uint64(queryHash[:8])

	// Add time component (changes every 30 seconds for reasonable rotation)
	timeComponent := uint64(time.Now().Unix() / 30)

	// Combine and mod by count
	combined := queryValue + timeComponent
	result := int(combined % uint64(count))

	// DEBUG: Add this logging
	log.Printf("RoundRobin DEBUG - queryValue: %d, timeComponent: %d, combined: %d, count: %d, result: %d",
		queryValue, timeComponent, combined, count, result)

	return result
}

// InitializeSchema creates the DNS records table using a schema file
func (s *PostgresStorage) InitializeSchema(ctx context.Context, schemaFilePath string) error {
	return s.pool.ExecSchemaFile(ctx, s.connectionName, schemaFilePath)
}
