# Crypto Monitoring Engine

A real-time, microservice-based cryptocurrency price monitoring system built with Go, Redis, and WebSockets.

The engine fetches high-frequency price data from the Binance API, routes it through a Redis message bus, and is deployed to a Kubernetes (K3s) cluster with full Prometheus observability, security hardening, and a **GitOps** automation pipeline.

## 🏗️ Architecture

The project is split into functional components for independent scaling:

1.  **Fetcher Service**: Maintains a robust WebSocket connection to Binance. It parses trade events (combined streams) and publishes them to a password-protected Redis Pub/Sub channel.
2.  **Redis Message Broker**: High-speed decoupling layer with mandatory authentication.
3.  **Processor Service**: Subscribes to Redis, calculates metrics, and exposes a Prometheus endpoint on port `8080`.

## 🛠️ Tech Stack

*   **Language**: Go (1.25.6+)
*   **Infrastructure**: Ansible & RHEL 9.7 (Homelab)
*   **Orchestration**: K3s (Kubernetes)
*   **CI/CD**: GitHub Actions & ArgoCD (GitOps)
*   **Registry**: GitHub Container Registry (GHCR)
*   **Observability**: Prometheus & Grafana
*   **Security**: Non-root container execution (UID 1000), K8s Secrets, Firewalld & SELinux.

---

## 📋 Prerequisites

Before you begin, ensure you have the following configured:

### 1. Local Tools (Mac)
- **Go 1.25+**: For local development.
- **Ansible**: `brew install ansible` (Used for provisioning).
- **Docker**: For local image testing.

### 2. Infrastructure Setup
- **RHEL 9.7 Server**: A physical or virtual machine with SSH access.
- **SSH Keys**: Your Mac's public key (`~/.ssh/id_rsa.pub`) must be in the server's `~/.ssh/authorized_keys`.
- **Sudo Access**: The user in `inventory.ini` must have passwordless sudo access on the RHEL server.

- **GHCR Token**: Create a **Personal Access Token (Classic)** with `write:packages` and `repo` scopes.
- **Repo Secret**: Add this token as a Repository Secret named **`GH_PAT`** in your GitHub repository (**Settings > Secrets and variables > Actions**).
- **Redis Secret**: Create a local file `ansible/redis-secret.yaml` by copying the template: `cp ansible/redis-secret.yaml.example ansible/redis-secret.yaml`. 
- **Base64 Encode**: Use `echo -n "your-password" | base64` to generate the password value for the manifest. This file is excluded from Git.

---

## 🛠️ Cluster Provisioning (First-Time Setup)

Before you can deploy the engine, you must prepare your RHEL 9.7 node and install K3s. This is automated via Ansible.

### 1. Configure Inventory
Edit `ansible/inventory.ini` and add your RHEL server's IP address:
```ini
[homelab]
node1 ansible_host=<YOUR_SERVER_IP> ansible_user=<YOUR_USERNAME>
```

### 2. Run the Playbook
From your Mac, run the following command to install K3s and configure the firewall:
```bash
cd ansible
ansible-playbook -i inventory.ini playbook.yml
```

### 3. What happens next?
The playbook will:
- **Configure Firewall**: Open ports `6443`, `10250`, etc., and enable masquerading.
- **Install K3s**: Install the lightweight Kubernetes distribution with SELinux support.
- **Generate Kubeconfig**: Fetch the cluster's internal configuration (`k3s.yaml`) to your Mac.
- **Patch IP**: Automatically replace `127.0.0.1` with your server's IP in the local file.
- **Secure Secrets**: Inject your local `ansible/redis-secret.yaml` directly into the cluster (bypassing Git).
- **Final Result**: You will see a new file `ansible/k3s-homelab.yaml` on your Mac. **This is your "passport" to the cluster.**

---

## 💻 Recommended Workflow: GitOps (Automated)

The project uses a professional "Push-to-Deploy" pipeline triggered from the `main` branch.

### 1. Continuous Integration (CI)
When you push code to `main`, **GitHub Actions** automatically builds the Go binaries and pushes a fresh Docker image to **GHCR**.

### 2. Continuous Deployment (CD)
**ArgoCD** runs in your cluster, monitors the `k8s/` directory, and automatically pulls the new image. 

---

## 🛠️ Legacy/Manual Workflow (For Debugging Only)

> [!WARNING]
> **Outdated**: This method manually overrides the GitOps sync and is only recommended for offline development or emergency debugging when you cannot reach GitHub.

### 1. Build and Sync Images Manually
If you need to bypass the automated pipeline, use the sync script to build for `amd64` and pipe the image directly into K3s over SSH:

```bash
# This builds the image locally for linux/amd64 and imports it into k3s manually
./scripts/sync-images.sh
```

### 2. Manual Deploy
After manual syncing, you must manually apply the manifests:

```bash
export KUBECONFIG=ansible/k3s-homelab.yaml
kubectl apply -f k8s/
```

---

## ⚙️ Configuration

You can customize the engine's behavior (symbols, Redis URL, roles) by modifying the environment variables in the following locations:

### 1. Local Development (Docker Compose)
Edit the `environment` section in [docker-compose.yml](file:///Users/jaychiu/Desktop/Projects/crypto-monitoring-engine-go/docker-compose.yml):
```yaml
environment:
  - APP_ROLE=fetcher
  - REDIS_URL=redis:6379
  - REDIS_PASSWORD=your_local_password
  - SYMBOLS=btcusdt,ethusdt,solusdt,bnbusdt
```

### 2. Production (Kubernetes / ArgoCD)
Edit the `Deployment` manifests in [k8s/crypto-engine.yaml](file:///Users/jaychiu/Desktop/Projects/crypto-monitoring-engine-go/k8s/crypto-engine.yaml) and then **push to main** to trigger an automatic update.
- **Symbols/Roles**: Update the `env` list in the deployment spec.
- **Passwords**: Update the **`ansible/redis-secret.yaml`** on your Mac and run the `argocd-setup.yml` playbook. **Never commit passwords to Git.**

---

## 🔍 Status & Debugging

### GitOps Dashboard (ArgoCD)
Monitor the real-time sync status and health of your applications:
- **URL**: http://<NODE_IP>:30081
- **Username**: `admin`
- **Password**: Run `kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo`

### Observability Dashboard (Grafana Loki)
Aggregate and search logs across all services:
- **URL**: http://<NODE_IP>:30082
- **Username**: `admin`
- **Password**: Run `kubectl get secret --namespace observability loki-observability-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo` to retrieve it.

### CLI Troubleshooting Commands

#### Check Cluster Health
```bash
# Note: The k3s-homelab.yaml file contains sensitive credentials and is NOT committed to Git. 
# It is generated locally on your machine during the Ansible setup phase.
export KUBECONFIG=ansible/k3s-homelab.yaml

# Check if the ArgoCD GitOps connection is healthy
kubectl get applications -n argocd

# Check all running pods and their nodes
kubectl get pods -o wide
```

#### Application Debugging
```bash
# View live logs for the Binance Fetcher
kubectl logs -f -l app=engine-fetcher

# View live logs for the Metrics Processor
kubectl logs -f -l app=engine-processor

# Check for resource/scheduling issues (useful if pod is "Pending")
kubectl describe pod -l app=engine-fetcher
```

#### Network & Metrics
```bash
# Verify the metrics endpoint is reachable from your Mac
curl http://<NODE_IP>:30080/metrics | grep crypto_price
```

---

## 🚀 Project Roadmap

- [x] **Phase 1**: Go WebSocket Project Skeleton
- [x] **Phase 2**: Redis Pub/Sub & Docker Integration
- [x] **Phase 3**: Homelab Provisioning (RHEL 9.7 & K3s)
- [x] **Phase 4**: Kubernetes Deployment (Manual Sync)
- [x] **Phase 5**: Security Hardening (Auth & Non-root)
- [x] **Phase 6**: Automated CI/CD (GitHub Actions & ArgoCD)
- [x] **Phase 7**: Full Observability (Loki/Promtail/Grafana)

