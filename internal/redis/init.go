package redis

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// Client is the global Redis client for backward compatibility
	Client *redis.Client

	// Context for Redis operations (package private)
	ctx = context.Background()

	// Map of named Redis clients
	clients = make(map[string]*redis.Client)

	// Mutex for thread-safe access to the clients map
	clientsMutex sync.RWMutex

	// Flag to track if cleanup hook is set
	cleanupSet bool
	cleanupMux sync.Mutex
)

// GetContext returns the context used for Redis operations
func GetContext() context.Context {
	return ctx
}

// Initialize creates a new Redis client with the default name
// Kept for backward compatibility
func Initialize(r string) *redis.Client {
	return NewClient("default", r, true)
}

// NewClient creates a new Redis client with the given name and address
func NewClient(name, address string, useExisting bool) *redis.Client {
	if address == "" {
		address = "localhost:6379"
	}

	// use an existing connection unless otherwise requested
	if useExisting {
		clientsMutex.RLock()
		if client, exists := clients[name]; exists {
			clientsMutex.RUnlock()
			return client
		}
		clientsMutex.RUnlock()
	}

	client := redis.NewClient(&redis.Options{
		Addr:            address,
		Password:        "",                             // no password by default
		DB:              0,                              // use default DB
		PoolSize:        10,                             // connection pool size
		MinIdleConns:    3,                              // minimum number of idle connections
		ConnMaxIdleTime: 240 * time.Second,              // how long connections stay idle
		DialTimeout:     time.Duration(2 * time.Second), // 2 second timeout for making connections
	})

	// Store in our clients map
	clientsMutex.Lock()
	clients[name] = client
	clientsMutex.Unlock()

	// Set as the default Client if this is the "default" client or first client
	if name == "default" || Client == nil {
		Client = client
	}

	// Ensure cleanup hook is set
	ensureCleanupHook()

	return client
}

// GetClient returns a Redis client by name
func GetClient(name string) *redis.Client {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	if client, exists := clients[name]; exists {
		return client
	}

	// Return the default client if the named one doesn't exist
	return Client
}

// Close closes a specific Redis client by name
func Close(name string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	if client, exists := clients[name]; exists {
		client.Close()
		delete(clients, name)

		// If we closed the default client, reset it to another one or nil
		if client == Client {
			if len(clients) > 0 {
				for _, c := range clients {
					Client = c
					break
				}
			} else {
				Client = nil
			}
		}
	}
}

// CloseAll closes all Redis clients
func CloseAll() {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for name, client := range clients {
		client.Close()
		delete(clients, name)
	}

	Client = nil
}

// ensureCleanupHook sets up the cleanup hook for graceful shutdown if not already set
func ensureCleanupHook() {
	cleanupMux.Lock()
	defer cleanupMux.Unlock()

	if !cleanupSet {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			CloseAll()
			os.Exit(0)
		}()
		cleanupSet = true
	}
}
