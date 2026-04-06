# Crypto Monitoring Engine

The goal is to build a system that fetches real-time price data from multiple sources, processes it in Go, and visualizes it on a dashboard, all while managed by a fully automated CI/CD and Infrastructure-as-Code (IaC) pipeline.

Go service pushes custom metrics (e.g., bitcoin_price_usd, arbitrage_opportunity_detected) to Prometheus. You then build a high-intensity dashboard in Grafana.

# Implementation Roadmap

## Phase 1: Go Application Foundation & WebSocket Client

- [ ] Create root directories (terraform, k8s, ansible, scripts, go, etc.).
- [ ] Implement a modular Go application (e.g., `cmd/engine`, `internal/client`) that fetches live BTC prices via Binance WebSockets.
- [ ] Include robust continuous connection logic (auto-reconnect with backoff).
- [ ] Implement structured logging (`slog`) and environment-based configuration.

## Phase 2: Local Docker Environment & Redis Integration

- [ ] Create a `docker-compose.yml` to run the Go app alongside a Redis instance locally.
- [ ] Update the Go app: one process (fetcher) publishes to Redis, while another (processor) reads the streams.

## Phase 3: Infrastructure Provisioning (Terraform & Ansible)

- [ ] Use Terraform to provision compute resources on AWS or homelab.
- [ ] Use Ansible to install K3s, required drivers, and prepare nodes.

## Phase 4: Kubernetes Deployment & GitOps

- [ ] Containerize the Go app and Redis components.
- [ ] Use Helm or Kustomize to template manifests.
- [ ] Deploy the microservices to the K3s cluster.

# Telemetry & Observability (OTel Stack)

*(Deploying via Helm community charts is recommended)*

## Phase 5: Add a Prometheus layer to scrape custom metrics and monitor the cluster.

## Phase 6: Add a Grafana layer to visualize the real-time data.

## Phase 7: Add a Loki layer to ingest and store the structured application logs.

## Phase 8: Add a Tempo layer for distributed tracing.
