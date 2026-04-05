# Crypto Monitoring Engine

A real-time, microservice-based cryptocurrency price monitoring system built with Go, Redis, and WebSockets.

The engine fetches high-frequency price data from the Binance API, routes it through a Redis message bus, and is designed to eventually be deployed to a Kubernetes cluster with a full OpenTelemetry observability stack.

## Architecture

The project is split into functional components to allow for independent scaling:

1.  **Fetcher Service**: Specialized in maintaining a robust WebSocket connection to Binance. It parses trade events and publishes them as JSON strings to a Redis Pub/Sub channel.
2.  **Redis Message Broker**: Serves as the high-speed decoupling layer. Multiple processors can listen to a single fetcher's feed.
3.  **Processor Service**: Subscribes to Redis and handles the data (currently logging to stdout, but eventually used for Prometheus metrics generation).

## Tech Stack

*   **Language**: Go (1.25.6+)
*   **Message Broker**: Redis
*   **Containerization**: Docker & Docker Compose
*   **Infrastructure (Planned)**: Terraform & Ansible
*   **Orchestration (Planned)**: K3s / Kubernetes
*   **Observability (Planned)**: Prometheus, Grafana, Loki, Tempo

## Getting Started

### Prerequisites

*   Docker Desktop
*   Go 1.25.6+ (for local development)

### Run Locally (Docker Compose)

The easiest way to see the engine in action is via Docker Compose:

```bash
docker compose up --build
```

This will start:
*   A `redis` container.
*   An `engine-fetcher` container.
*   An `engine-processor` container.

You should see live `BTCUSDT` price updates flowing through the logs.

## Project Roadmap

- [x] Phase 1: Go WebSocket Project Skeleton
- [x] Phase 2: Redis Pub/Sub & Docker Integration
- [ ] Phase 3: Infrastructure Provisioning (Terraform & Ansible)
- [ ] Phase 4: Kubernetes Deployment
- [ ] Phase 5: Prometheus Metrics & Grafana Dashboard
- [ ] Phase 6: Full Observability (Loki/Tempo)
