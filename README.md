# Crypto Monitoring Engine

A real-time, microservice-based cryptocurrency price monitoring system built with Go, Redis, and WebSockets.

The engine fetches high-frequency price data from the Binance API, routes it through a Redis message bus, and is deployed to a Kubernetes (K3s) cluster with full Prometheus observability and security hardening.

## 🏗️ Architecture

The project is split into functional components for independent scaling:

1.  **Fetcher Service**: Maintains a robust WebSocket connection to Binance. It parses trade events (combined streams) and publishes them to a password-protected Redis Pub/Sub channel.
2.  **Redis Message Broker**: High-speed decoupling layer with mandatory authentication.
3.  **Processor Service**: Subscribes to Redis, calculates metrics, and exposes a Prometheus endpoint on port `8080`.

## 🛠️ Tech Stack

*   **Language**: Go (1.25.6+)
*   **Message Broker**: Redis 7 (Authenticated)
*   **Infrastructure**: Ansible & RHEL 9.7 (Homelab)
*   **Orchestration**: K3s (Kubernetes)
*   **Observability**: Prometheus & Grafana
*   **Security**: Non-root container execution (UID 1000), K8s Secrets, Firewalld & SELinux.

---

## 💻 Development Workflow (Homelab)

If you are developing locally on a Mac and deploying to a RHEL homelab, use the following "Sync & Apply" workflow:

### 1. Build and Sync Images
Since we use a private homelab without a public registry, use the sync script to build for `amd64` and pipe the image directly into K3s over SSH:

```bash
# This builds the image locally for linux/amd64 and imports it into k3s
./scripts/sync-images.sh
```

### 2. Deploy to Cluster
After syncing the image, apply the Kubernetes manifests:

```bash
export KUBECONFIG=ansible/k3s-homelab.yaml
kubectl apply -f k8s/
```

---

## 🔍 Verification & Debugging

Use these commands to ensure the engine is healthy in the cluster:

### Check Pod Status
```bash
kubectl get pods -o wide
```
*Note: All pods should be in the `Running` state.*

### Verify Security (Non-Root Check)
Confirm the application is running as `appuser` (UID 1000), not `root`:
```bash
kubectl exec deployment/engine-processor -- id
```

### View Live Logs
```bash
# Watch the fetcher connect to Binance
kubectl logs -f -l app=engine-fetcher

# Watch the processor receive Redis updates
kubectl logs -f -l app=engine-processor
```

### Check Live Metrics
Verify that Prometheus can scrape the price data from your Mac:
```bash
curl http://192.168.1.200:30080/metrics
```

---

## 🚀 Project Roadmap

- [x] **Phase 1**: Go WebSocket Project Skeleton
- [x] **Phase 2**: Redis Pub/Sub & Docker Integration
- [x] **Phase 3**: Homelab Provisioning (RHEL 9.7 & K3s)
- [x] **Phase 4**: Kubernetes Deployment (Manual Sync)
- [x] **Phase 5**: Security Hardening (Auth & Non-root)
- [/] **Phase 6**: Automated CI/CD (GitHub Actions & ArgoCD)
- [ ] **Phase 7**: Full Observability (Loki/Tempo)
