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
	"errantdns.io/internal/logging"
	"errantdns.io/internal/pgsqlpool"
	"errantdns.io/internal/redis"
	"errantdns.io/internal/storage"
)

func main() {
	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Initialize logging EARLY - before any other operations
	loggingConfig := &logging.Config{
		Level:           logging.LogLevel(cfg.Logging.Level),
		Directory:       cfg.Logging.Directory,
		AppLogFile:      cfg.Logging.AppLogFile,
		QueryLogFile:    cfg.Logging.QueryLogFile,
		ErrorLogFile:    cfg.Logging.ErrorLogFile,
		EnableConsole:   cfg.Logging.EnableConsole,
		QuerySampleRate: cfg.Logging.QuerySampleRate,
		BufferSize:      cfg.Logging.BufferSize,
	}

	if err := logging.Initialize(loggingConfig); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}

	// Now use the new logging system
	logging.Info("main", "ErrantDNS server starting",
		"version", "1.0.0",
		"dns_port", cfg.DNSPort,
		"cache_enabled", cfg.Cache.Enabled,
		"redis_enabled", cfg.Redis.Enabled)

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

		if cfg.Redis.Enabled {
			// Initialize Redis client
			log.Printf("Initializing Redis connection to %s", cfg.Redis.Address)
			redis.NewClient(cfg.Redis.ClientName, cfg.Redis.Address, false)

			// Test Redis connection
			if err := redis.PingClient(cfg.Redis.ClientName); err != nil {
				log.Fatalf("Failed to connect to Redis: %v", err)
			}
			log.Printf("Connected to Redis at %s", cfg.Redis.Address)

			// Three-tier caching: Memory → Redis → PostgreSQL
			finalStorage = storage.NewRedisCacheStorage(pgStorage, memCache, cfg.Redis.ClientName, "errantdns:", cfg.Priority.TieBreaker)
			log.Printf("Three-tier cache enabled: Memory → Redis → PostgreSQL")
		} else {
			// Two-tier caching: Memory → PostgreSQL
			finalStorage = storage.NewCachedStorage(pgStorage, memCache, cfg.Priority.TieBreaker)
			log.Printf("Two-tier cache enabled: Memory → PostgreSQL")
		}

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
	go reportStats(ctx, dnsServer, finalStorage, cfg)

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

	if cfg.Redis.Enabled {
		redis.Close(cfg.Redis.ClientName)
		log.Printf("Redis connection closed")
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

	defer func() {
		if err := logging.GetLogger().Close(); err != nil {
			logging.Error("main", "Failed to close logging", err)
		}
	}()
}

// reportStats periodically reports server and cache statistics
func reportStats(ctx context.Context, dnsServer *dns.Server, storage storage.Storage, cfg *config.Config) {
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

			log.Printf("Query Types - A: %d, AAAA: %d, CNAME: %d, MX: %d, TXT: %d, NS: %d, SOA: %d, PTR: %d, SRV: %d, CAA: %d, Other: %d",
				dnsStats.TypeA, dnsStats.TypeAAAA, dnsStats.TypeCNAME,
				dnsStats.TypeMX, dnsStats.TypeTXT, dnsStats.TypeNS, dnsStats.TypeSOA, dnsStats.TypePTR, dnsStats.TypeSRV, dnsStats.TypeCAA, dnsStats.TypeOther)

			// Try to get cache stats using a type assertion that will work
			// We need to check if the storage has a GetCacheStats method
			type CacheStatsProvider interface {
				GetCacheStats() cache.Stats
			}

			// Cache statistics reporting
			if cfg.Cache.Enabled {
				if cfg.Redis.Enabled {
					// Three-tier cache stats
					log.Printf("Cache Status: Three-tier (Memory + Redis)")

					// Try to get memory cache stats
					type MemoryCacheProvider interface {
						GetCacheStats() cache.Stats
					}

					// For now, log that we need to implement Redis-specific stats
					log.Printf("L1 Cache (Memory): Stats collection needs implementation")
					log.Printf("L2 Cache (Redis): Connected to %s", cfg.Redis.Address)

					// Check Redis connectivity
					if err := redis.PingClient(cfg.Redis.ClientName); err != nil {
						log.Printf("L2 Cache (Redis): Connection error - %v", err)
					} else {
						log.Printf("L2 Cache (Redis): Connection healthy")
					}
				} else {
					// Two-tier cache stats
					log.Printf("Cache Status: Two-tier (Memory only)")

					type CacheStatsProvider interface {
						GetCacheStats() cache.Stats
					}

					if cacheProvider, ok := storage.(CacheStatsProvider); ok {
						cacheStats := cacheProvider.GetCacheStats()
						log.Printf("L1 Cache Stats - Entries: %d, Hits: %d, Misses: %d, Hit Rate: %.2f%%, Evictions: %d",
							cacheStats.Entries, cacheStats.Hits, cacheStats.Misses,
							cacheStats.HitRate, cacheStats.Evictions)
					}
				}
			} else {
				log.Printf("Cache Status: Disabled (Direct database access)")
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
