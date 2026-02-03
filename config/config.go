package config

import (
	"os"
	"strings"
)

type Role string

const (
	RolePrimary Role = "primary"
	RoleReplica Role = "replica"
)

type Config struct {
	Role        Role
	Port        string
	DataDir     string   // Directory for persistent storage
	ReplicaURLs []string // URLs of replicas (used by primary)
	PrimaryURL  string   // URL of primary (used by replicas)
}

func Load() *Config {
	cfg := &Config{
		Role:       RoleReplica,
		Port:       getEnv("PORT", "8080"),
		DataDir:    getEnv("DATA_DIR", "./data"),
		PrimaryURL: getEnv("PRIMARY_URL", ""),
	}

	// Determine role
	role := strings.ToLower(getEnv("ROLE", ""))
	if role == "primary" {
		cfg.Role = RolePrimary
	}

	// Parse replica URLs
	replicaURLs := getEnv("REPLICA_URLS", "")
	if replicaURLs != "" {
		cfg.ReplicaURLs = strings.Split(replicaURLs, ",")
		// Trim whitespace
		for i, url := range cfg.ReplicaURLs {
			cfg.ReplicaURLs[i] = strings.TrimSpace(url)
		}
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) IsPrimary() bool {
	return c.Role == RolePrimary
}
