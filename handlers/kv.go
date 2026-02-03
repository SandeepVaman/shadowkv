package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"kvstore/replication"
	"kvstore/store"
)

type KVHandler struct {
	store      *store.Store
	replicator *replication.Replicator
	isPrimary  bool
	primaryURL string
}

func NewKVHandler(s *store.Store, r *replication.Replicator, isPrimary bool, primaryURL string) *KVHandler {
	return &KVHandler{
		store:      s,
		replicator: r,
		isPrimary:  isPrimary,
		primaryURL: primaryURL,
	}
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *KVHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract key from path: /kv/{key}
	path := strings.TrimPrefix(r.URL.Path, "/kv/")
	key := strings.TrimPrefix(path, "/")

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, key)
	case http.MethodPut:
		h.handlePut(w, r, key)
	case http.MethodDelete:
		h.handleDelete(w, r, key)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "method not allowed"})
	}
}

func (h *KVHandler) handleGet(w http.ResponseWriter, r *http.Request, key string) {
	w.Header().Set("Content-Type", "application/json")

	if key == "" {
		// Return all keys
		keys := h.store.Keys()
		json.NewEncoder(w).Encode(map[string][]string{"keys": keys})
		return
	}

	value, exists := h.store.Get(key)
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "key not found"})
		return
	}

	json.NewEncoder(w).Encode(KeyValue{Key: key, Value: value})
}

func (h *KVHandler) handlePut(w http.ResponseWriter, r *http.Request, key string) {
	w.Header().Set("Content-Type", "application/json")

	// Only primary can accept writes
	if !h.isPrimary {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "writes only accepted on primary node",
		})
		return
	}

	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "key is required"})
		return
	}

	// Read value from body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to read body"})
		return
	}

	var value string
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var kv KeyValue
		if err := json.Unmarshal(body, &kv); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid JSON"})
			return
		}
		value = kv.Value
	} else {
		// Plain text body is the value
		value = string(body)
	}

	// Store locally
	if err := h.store.Set(key, value); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "failed to store value"})
		return
	}

	// Replicate to replicas
	if h.replicator != nil {
		h.replicator.ReplicateSet(key, value)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(KeyValue{Key: key, Value: value})
}

func (h *KVHandler) handleDelete(w http.ResponseWriter, r *http.Request, key string) {
	w.Header().Set("Content-Type", "application/json")

	// Only primary can accept writes
	if !h.isPrimary {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "writes only accepted on primary node",
		})
		return
	}

	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "key is required"})
		return
	}

	existed := h.store.Delete(key)

	// Replicate to replicas
	if h.replicator != nil {
		h.replicator.ReplicateDelete(key)
	}

	if !existed {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "key not found"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"deleted": key})
}
