package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"kvstore/config"
	"kvstore/store"
)

type HealthHandler struct {
	store *store.Store
	cfg   *config.Config
}

func NewHealthHandler(s *store.Store, cfg *config.Config) *HealthHandler {
	return &HealthHandler{
		store: s,
		cfg:   cfg,
	}
}

type HealthResponse struct {
	Status   string `json:"status"`
	Hostname string `json:"hostname,omitempty"`
}

type ReadyResponse struct {
	Status   string `json:"status"`
	Role     string `json:"role"`
	Keys     int    `json:"keys"`
	Hostname string `json:"hostname,omitempty"`
}

type RoleResponse struct {
	Role     string `json:"role"`
	Hostname string `json:"hostname,omitempty"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	hostname, _ := os.Hostname()
	json.NewEncoder(w).Encode(HealthResponse{
		Status:   "healthy",
		Hostname: hostname,
	})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	hostname, _ := os.Hostname()
	json.NewEncoder(w).Encode(ReadyResponse{
		Status:   "ready",
		Role:     string(h.cfg.Role),
		Keys:     h.store.Len(),
		Hostname: hostname,
	})
}

func (h *HealthHandler) Role(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	hostname, _ := os.Hostname()
	json.NewEncoder(w).Encode(RoleResponse{
		Role:     string(h.cfg.Role),
		Hostname: hostname,
	})
}
