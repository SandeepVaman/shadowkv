# Distributed Key-Value Store

A simple distributed key-value store with primary-replica architecture, designed for learning Kubernetes StatefulSet deployments.

## Architecture

```
                    ┌─────────────────────────────────────────────────────┐
                    │                    Clients                          │
                    └─────────────────────────────────────────────────────┘
                                │                       │
                                │ Writes                │ Reads
                                ▼                       ▼
                    ┌───────────────────┐   ┌───────────────────┐
                    │     Primary       │   │  Load Balancer    │
                    │     (pod-0)       │   │  (all replicas)   │
                    │                   │   └───────────────────┘
                    │  - Accepts writes │
                    │  - Accepts reads  │
                    │  - Replicates to  │
                    │    all replicas   │
                    └─────────┬─────────┘
                              │
              ┌───────────────┼───────────────┐
              │ Replication   │               │
              ▼               ▼               ▼
    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
    │  Replica 1   │ │  Replica 2   │ │  Replica N   │
    │   (pod-1)    │ │   (pod-2)    │ │   (pod-N)    │
    │              │ │              │ │              │
    │ - Reads only │ │ - Reads only │ │ - Reads only │
    │ - Receives   │ │ - Receives   │ │ - Receives   │
    │   replicated │ │   replicated │ │   replicated │
    │   data       │ │   data       │ │   data       │
    └──────────────┘ └──────────────┘ └──────────────┘
```

### Key Points

- **Single Primary**: One node (pod-0) handles all writes
- **Multiple Replicas**: Read-only nodes that receive replicated data
- **Async Replication**: Primary pushes changes to replicas (fire-and-forget)
- **Eventual Consistency**: Replicas may lag slightly behind primary
- **No Leader Election**: Primary is determined by hostname (pod-0 is always primary)

## API Endpoints

| Method | Endpoint | Description | Served By |
|--------|----------|-------------|-----------|
| GET | `/kv/{key}` | Get value for key | All nodes |
| GET | `/kv/` | List all keys | All nodes |
| PUT | `/kv/{key}` | Set key-value (body = value) | Primary only |
| DELETE | `/kv/{key}` | Delete key | Primary only |
| GET | `/health` | Liveness probe | All nodes |
| GET | `/ready` | Readiness probe (includes role & key count) | All nodes |
| GET | `/role` | Get node role (primary/replica) | All nodes |

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `ROLE` | Node role: `primary` or `replica` | `replica` |
| `PORT` | HTTP listen port | `8080` |
| `DATA_DIR` | Directory for persistent storage (BoltDB) | `./data` |
| `REPLICA_URLS` | Comma-separated replica URLs (primary only) | empty |
| `PRIMARY_URL` | Primary URL (replica only, for reference) | empty |

**Persistence**: Data is stored in BoltDB at `$DATA_DIR/kvstore.db`. Each node maintains its own database file.

## Running Locally

### Prerequisites

- Go 1.21 or higher

Verify your Go version:
```bash
go version
```

### Build

```bash
go build -o kvstore .
```

### Run Primary and Replica

**Terminal 1 - Primary:**
```bash
export PORT=8081
export ROLE=primary
export DATA_DIR=./data/primary
export REPLICA_URLS=http://localhost:8082
./kvstore
```

**Terminal 2 - Replica:**
```bash
export PORT=8082
export ROLE=replica
export DATA_DIR=./data/replica
export PRIMARY_URL=http://localhost:8081
./kvstore
```

### Test

```bash
# Write to primary
curl -X PUT http://localhost:8081/kv/name -d "alice"

# Read from primary
curl http://localhost:8081/kv/name

# Read from replica (should return same value)
curl http://localhost:8082/kv/name

# Try writing to replica (should fail)
curl -X PUT http://localhost:8082/kv/name -d "bob"

# List all keys
curl http://localhost:8081/kv/

# Delete a key
curl -X DELETE http://localhost:8081/kv/name

# Check node roles
curl http://localhost:8081/role
curl http://localhost:8082/role

# Health checks
curl http://localhost:8081/health
curl http://localhost:8081/ready
```

---

## Assignment: Kubernetes Deployment

### Objective

Write a Dockerfile and Helm charts to deploy this distributed key-value store on Kubernetes (minikube or any k8s platform).

### Requirements

1. **Dockerfile** to containerize the application

2. **Helm Chart** with:
   - **StatefulSet** for the kvstore pods
     - Pod with ordinal 0 should be primary
     - Remaining pods should be replicas
     - Configure environment variables appropriately
   - **Headless Service** for stable network identities
     - Enables DNS names like `kvstore-0.kvstore.default.svc.cluster.local`
   - **ClusterIP Service** (optional) for load-balanced reads
   - **Proper probes**
     - Liveness probe using `/health`
     - Readiness probe using `/ready`
   - **Configurable replica count** via Helm values

### Hints

- StatefulSet pods get predictable hostnames: `<statefulset-name>-<ordinal>`
- With a headless service, each pod gets a DNS entry: `<pod-name>.<service-name>.<namespace>.svc.cluster.local`
- The primary needs to know replica URLs - consider how to construct these dynamically
- Pod ordinal can be extracted from hostname to determine role

### Verification

After deploying to minikube:

```bash
# Port-forward to primary
kubectl port-forward pod/kvstore-0 8081:8080 &

# Port-forward to replica
kubectl port-forward pod/kvstore-1 8082:8080 &

# Test write to primary
curl -X PUT http://localhost:8081/kv/test -d "hello"

# Test read from replica
curl http://localhost:8082/kv/test

# Check roles
curl http://localhost:8081/role
curl http://localhost:8082/role
```
