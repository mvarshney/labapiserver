# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lab API Server is a Go-based HTTP API server with comprehensive OpenTelemetry observability (metrics, traces, logs). It's deployed to Kubernetes using ArgoCD GitOps with a full observability stack.
This is learning project for me, where I can see the different aspects of deploying and operating 
microservices handson. This is not meant for production. The goals are:
1. Using idiomatic code and patterns
2. Hands on learning

## Development Commands

### Building and Running Locally

```bash
# Run the server locally
go run cmd/server/main.go

# Build binary
go build -o bin/labapiserver cmd/server/main.go

# Run with OTEL collector endpoint
OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317" OTEL_COLLECTOR_ENDPOINT="localhost:4317" go run cmd/server/main.go
```

### Container Building

This project uses [Ko](https://ko.build/) for building Go containers:

```bash
# Build and push to registry (configured in .ko.yaml)
ko build ./cmd/server

# Build for multiple platforms
ko build --bare --platform=linux/amd64,linux/arm64 ./cmd/server

# Build locally without pushing
ko build --local ./cmd/server
```

Configuration in `.ko.yaml` uses `cgr.dev/chainguard/static:latest` as the base image.

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Architecture

### Code Structure

```
cmd/server/          - Main application entry point
internal/
  handlers/          - HTTP request handlers
    salestax/        - Sales tax calculation endpoint
    metrics/         - Memory monitoring and metrics
  middleware/        - HTTP middleware (tracing, observability)
  metrics/           - OTEL metrics initialization and instruments
  tracing/           - OTEL tracing initialization
  config/            - Configuration management
pkg/
  health/            - Health check endpoint
```

### Observability Stack

The application implements comprehensive OpenTelemetry instrumentation:

**Tracing** (`internal/tracing/`):
- Configurable exporter: stdout (dev) or OTLP gRPC (prod)
- Environment variable: `OTEL_EXPORTER_OTLP_ENDPOINT`
- Middleware in `internal/middleware/tracing.go` wraps handlers
- Trace context propagation via HTTP headers

**Metrics** (`internal/metrics/`):
- Push-based metrics to OTLP collector (10s interval)
- Environment variable: `OTEL_EXPORTER_OTLP_ENDPOINT`
- Instruments defined:
  - Counters: `http.requests.total`, `http.errors.total`
  - UpDownCounter: `http.active_connections`
  - Histograms: `http.request.duration`, `http.response.size`
  - Gauge: `runtime.memory.heap.alloc`
- Middleware in `internal/middleware/observability.go` wraps handlers

**Structured Logging**:
- Uses Go's `log/slog` with JSON output
- Includes trace IDs in log entries for correlation

### Middleware Chain

Handlers are wrapped with both tracing and observability middleware:

```go
mux.Handle("/endpoint",
    middleware.TracingMiddleware("name")(
        middleware.ObservabilityMiddleware("name")(handler)))
```

This ensures:
1. Tracing context is established first
2. Metrics collection happens within the trace span
3. Both middleware layers coordinate on status code capture

### Deployment Architecture

**GitOps with ArgoCD**:
- Root app pattern: `iac/argocd/root-app.yaml` deploys all applications
- Individual apps in `iac/argocd/applications/`
- Helm charts in `iac/apps/`

**Infrastructure Components**:
- **Kong**: API Gateway with Ingress controller
- **Prometheus**: Metrics storage and querying
- **Loki**: Log aggregation
- **Tempo**: Distributed tracing backend
- **Vault**: Secrets management
- **OTel Collector**: Telemetry pipeline (receives OTLP, exports to backends)
- **Fluent-bit**: Log collection
- **Promtail**: Loki log shipper
- **Locust**: Load testing

**Application Deployment** (`iac/apps/labapiserver/`):
- 2 replicas by default
- Ingress via Kong at `api.local/api`
- Health checks on `/health`
- Resource limits: 500m CPU, 512Mi memory
- Runs as non-root (UID 1000) with read-only root filesystem

### Environment Variables

Key environment variables for the application:

- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP endpoint for metrics (e.g., `otel-collector:4317`)
- `OTEL_COLLECTOR_ENDPOINT`: OTLP endpoint for traces
- `OTEL_EXPORTER_OTLP_PROTOCOL`: Set to `grpc`
- `OTEL_SERVICE_NAME`: Service identifier for telemetry
- `PORT`: HTTP server port (default: 8080)
- `LOG_LEVEL`: Logging level

## Load Testing

Locust configuration in `iac/apps/locust/`:

```bash
# Access Locust UI (when deployed)
kubectl port-forward -n default svc/locust 8089:8089

# Test endpoints:
# - /salestax (weighted 3x)
# - /health (weighted 1x)
```

## CI/CD

GitHub Actions workflow (`.github/workflows/build-and-push.yaml`):
- Triggers on push to main or tags
- Builds multi-arch images (amd64, arm64) using Ko
- Pushes to GitHub Container Registry (ghcr.io)
- Tags: `latest`, `<git-sha>`, and `<tag-name>` for releases

## Adding New Endpoints

When adding a new HTTP endpoint:

1. Create handler in `internal/handlers/<name>/`
2. Implement handler with proper error handling and structured logging
3. Use `middleware.RecordError(ctx, handlerName, errorType)` for error metrics
4. Register in `cmd/server/main.go` with both middleware wrappers:
   ```go
   mux.Handle("/endpoint",
       middleware.TracingMiddleware("endpoint-name")(
           middleware.ObservabilityMiddleware("endpoint-name")(handler)))
   ```
5. Include trace ID in logs: `trace.SpanFromContext(ctx).SpanContext().TraceID().String()`

## Kubernetes Operations

```bash
# Apply ArgoCD root app
kubectl apply -f iac/argocd/root-app.yaml

# Check application status
kubectl get applications -n argocd

# View logs
kubectl logs -n default -l app.kubernetes.io/name=labapiserver -f

# Port-forward to service
kubectl port-forward -n default svc/labapiserver 8080:80
```
