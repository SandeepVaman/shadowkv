package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"kvstore/replication"
	"kvstore/store"
)

type ReplicateHandler struct {
	store *store.Store
}

func NewReplicateHandler(s *store.Store) *ReplicateHandler {
	return &ReplicateHandler{
		store: s,
	}
}

func (h *ReplicateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var cmd replication.Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		log.Printf("Failed to decode replication command: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch cmd.Operation {
	case replication.OpSet:
		if err := h.store.Set(cmd.Key, cmd.Value); err != nil {
			log.Printf("Failed to replicate SET %s: %v", cmd.Key, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("Replicated SET %s=%s", cmd.Key, cmd.Value)
	case replication.OpDelete:
		h.store.Delete(cmd.Key)
		log.Printf("Replicated DELETE %s", cmd.Key)
	default:
		log.Printf("Unknown replication operation: %s", cmd.Operation)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
