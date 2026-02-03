package replication

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type Operation string

const (
	OpSet    Operation = "SET"
	OpDelete Operation = "DELETE"
)

type Command struct {
	Operation Operation `json:"operation"`
	Key       string    `json:"key"`
	Value     string    `json:"value,omitempty"`
}

type Replicator struct {
	replicaURLs []string
	client      *http.Client
	mu          sync.RWMutex
}

func New(replicaURLs []string) *Replicator {
	return &Replicator{
		replicaURLs: replicaURLs,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (r *Replicator) SetReplicaURLs(urls []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.replicaURLs = urls
}

func (r *Replicator) GetReplicaURLs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.replicaURLs
}

func (r *Replicator) ReplicateSet(key, value string) {
	cmd := Command{
		Operation: OpSet,
		Key:       key,
		Value:     value,
	}
	r.replicate(cmd)
}

func (r *Replicator) ReplicateDelete(key string) {
	cmd := Command{
		Operation: OpDelete,
		Key:       key,
	}
	r.replicate(cmd)
}

func (r *Replicator) replicate(cmd Command) {
	r.mu.RLock()
	urls := make([]string, len(r.replicaURLs))
	copy(urls, r.replicaURLs)
	r.mu.RUnlock()

	if len(urls) == 0 {
		return
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		log.Printf("Failed to marshal replication command: %v", err)
		return
	}

	// Fire-and-forget async replication to all replicas
	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(replicaURL string) {
			defer wg.Done()
			r.sendToReplica(replicaURL, data)
		}(url)
	}

	// Optional: wait for replication to complete
	// For true async, remove this wg.Wait()
	wg.Wait()
}

func (r *Replicator) sendToReplica(replicaURL string, data []byte) {
	url := replicaURL + "/internal/replicate"
	resp, err := r.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("Failed to replicate to %s: %v", replicaURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Replication to %s returned status %d", replicaURL, resp.StatusCode)
	}
}
