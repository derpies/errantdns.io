// cmd/dns-server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"errantdns.io/internal/cache"
	"errantdns.io/internal/config"
	"errantdns.io/internal/dns"
	"errantdns.io/internal/pgsqlpool"
	"errantdns.io/internal/storage"
)

func main() {
	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Printf("Starting ErrantDNS server on port %s", cfg.DNSPort)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database pool
	pool := pgsqlpool.NewPool()

	// Create storage layer
	storageConfig := &storage.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	}

	pgStorage, err := storage.NewPostgresStorage(ctx, pool, cfg.Database.ConnectionName, storageConfig, cfg.Priority.TieBreaker)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	log.Printf("Connected to PostgreSQL database at %s:%d/%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// Create cache layer if enabled
	var finalStorage storage.Storage = pgStorage

	if cfg.Cache.Enabled {
		cacheConfig := &cache.Config{
			MaxEntries:      cfg.Cache.MaxEntries,
			CleanupInterval: cfg.Cache.CleanupInterval,
		}

		memCache := cache.NewMemoryCache(cacheConfig)
		finalStorage = storage.NewCachedStorage(pgStorage, memCache, cfg.Priority.TieBreaker)

		log.Printf("Cache enabled: max entries=%d, cleanup interval=%v",
			cfg.Cache.MaxEntries, cfg.Cache.CleanupInterval)
	} else {
		log.Printf("Cache disabled")
	}

	// Test storage health
	if err := finalStorage.Health(ctx); err != nil {
		log.Fatalf("Storage health check failed: %v", err)
	}

	log.Printf("Storage layer initialized successfully")

	// Create DNS server
	dnsConfig := &dns.Config{
		Port:          cfg.DNSPort,
		UDPTimeout:    5 * time.Second,
		TCPTimeout:    10 * time.Second,
		MaxConcurrent: cfg.MaxConcurrentQueries,
	}

	dnsServer := dns.NewServer(finalStorage, dnsConfig)

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start DNS server in background
	go func() {
		if err := dnsServer.Start(ctx); err != nil {
			log.Printf("DNS server error: %v", err)
			cancel()
		}
	}()

	// Start statistics reporting
	go reportStats(ctx, dnsServer, finalStorage)

	// Wait for shutdown signal
	<-sigChan
	log.Printf("Received shutdown signal, starting graceful shutdown...")

	// Cancel context to signal shutdown
	cancel()

	// Give servers time to shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown DNS server
	if err := dnsServer.Stop(); err != nil {
		log.Printf("Error during DNS server shutdown: %v", err)
	}

	// Close storage
	if err := finalStorage.Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	// Close database pool
	if err := pool.Close(); err != nil {
		log.Printf("Error closing database pool: %v", err)
	}

	select {
	case <-shutdownCtx.Done():
		log.Printf("Shutdown timeout exceeded")
	default:
		log.Printf("ErrantDNS server shutdown completed")
	}
}

// reportStats periodically reports server and cache statistics
func reportStats(ctx context.Context, dnsServer *dns.Server, storage storage.Storage) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get DNS server stats
			dnsStats := dnsServer.GetStats()

			log.Printf("DNS Stats - Queries: %d, Answered: %d, NXDOMAIN: %d, Errors: %d",
				dnsStats.QueriesReceived, dnsStats.QueriesAnswered,
				dnsStats.QueriesNXDomain, dnsStats.QueriesError)

			log.Printf("Query Types - A: %d, AAAA: %d, CNAME: %d, MX: %d, TXT: %d, NS: %d, SOA: %d, PTR: %d, SRV: %d, Other: %d",
				dnsStats.TypeA, dnsStats.TypeAAAA, dnsStats.TypeCNAME,
				dnsStats.TypeMX, dnsStats.TypeTXT, dnsStats.TypeNS, dnsStats.TypeSOA, dnsStats.TypePTR, dnsStats.TypeSRV, dnsStats.TypeOther)

			// Try to get cache stats using a type assertion that will work
			// We need to check if the storage has a GetCacheStats method
			type CacheStatsProvider interface {
				GetCacheStats() cache.Stats
			}

			if cacheProvider, ok := storage.(CacheStatsProvider); ok {
				cacheStats := cacheProvider.GetCacheStats()
				log.Printf("Cache Stats - Entries: %d, Hits: %d, Misses: %d, Hit Rate: %.2f%%, Evictions: %d",
					cacheStats.Entries, cacheStats.Hits, cacheStats.Misses,
					cacheStats.HitRate, cacheStats.Evictions)
			}
		}
	}
}

// printStartupInfo displays configuration information at startup
func printStartupInfo(cfg *config.Config) {
	fmt.Printf(`
ErrantDNS Server Starting
========================
DNS Port: %s
Database: %s:%d/%s (connection: %s)
Cache: %s (max entries: %d)
Max Concurrent Queries: %d
Log Level: %s

`,
		cfg.DNSPort,
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, cfg.Database.ConnectionName,
		func() string {
			if cfg.Cache.Enabled {
				return "Enabled"
			}
			return "Disabled"
		}(),
		cfg.Cache.MaxEntries,
		cfg.MaxConcurrentQueries,
		cfg.LogLevel,
	)
}
