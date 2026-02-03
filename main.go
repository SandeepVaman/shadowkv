package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"kvstore/config"
	"kvstore/handlers"
	"kvstore/replication"
	"kvstore/store"
)

func main() {
	// Load configuration
	cfg := config.Load()

	hostname, _ := os.Hostname()
	log.Printf("Starting kvstore on %s", hostname)
	log.Printf("Role: %s", cfg.Role)
	log.Printf("Port: %s", cfg.Port)
	log.Printf("Data directory: %s", cfg.DataDir)

	// Initialize store
	kvStore, err := store.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer kvStore.Close()

	// Initialize replicator (only used by primary)
	var replicator *replication.Replicator
	if cfg.IsPrimary() {
		replicator = replication.New(cfg.ReplicaURLs)
		log.Printf("Replica URLs: %v", cfg.ReplicaURLs)
	}

	// Initialize handlers
	kvHandler := handlers.NewKVHandler(kvStore, replicator, cfg.IsPrimary(), cfg.PrimaryURL)
	healthHandler := handlers.NewHealthHandler(kvStore, cfg)
	replicateHandler := handlers.NewReplicateHandler(kvStore)

	// Setup routes
	mux := http.NewServeMux()

	// KV operations
	mux.Handle("/kv/", kvHandler)
	mux.Handle("/kv", kvHandler)

	// Health endpoints
	mux.HandleFunc("/health", healthHandler.Health)
	mux.HandleFunc("/ready", healthHandler.Ready)
	mux.HandleFunc("/role", healthHandler.Role)

	// Internal replication endpoint
	mux.Handle("/internal/replicate", replicateHandler)

	// Create server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: loggingMiddleware(mux),
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		server.Close()
	}()

	// Start server
	log.Printf("Server listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
	log.Println("Server stopped")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
