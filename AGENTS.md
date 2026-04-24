# Agent Guide — frontend-asset-proxy

This document provides AI agents with the context and conventions needed to work effectively in this repository.

## Project Overview

A Go HTTP reverse proxy that serves frontend assets from S3-compatible object storage (AWS S3 or MinIO). It is part of the Frontend Operator (FEO) initiative for object storage-based push cache in the Red Hat Hybrid Cloud Console (HCC) platform.

**Key capabilities:**

| Feature | Description |
|---------|-------------|
| S3 streaming | Proxies `GetObject` responses from S3/MinIO via AWS SDK v2 |
| SPA routing | Falls back to `index.html` on 404/403 for single-page app support |
| Conditional requests | Supports `Range`, `If-None-Match`, `If-Modified-Since` headers |
| Health checks | `/healthz` endpoint for Kubernetes liveness probes |
| TLS support | Optional TLS via cert/key environment variables |

## Documentation Index

| Document | Description |
|----------|-------------|
| [docs/security-guidelines.md](docs/security-guidelines.md) | Credential handling, container security, TLS, request safety |
| [docs/testing-guidelines.md](docs/testing-guidelines.md) | Integration test patterns, Go unit test conventions |

## Repository Structure

```
frontend-asset-proxy/
  cmd/
    proxy/
      main.go                # HTTP server, routing, graceful shutdown
  internal/
    config/
      config.go              # Environment variable parsing, defaults
    logger/
      logger.go              # Structured logging, chi + AWS SDK integration
    s3/
      s3.go                  # S3 client, proxy streaming, error mapping
  .tekton/                   # Konflux/Tekton CI pipeline definitions
  Dockerfile                 # Multi-stage UBI9 container build
  Makefile                   # Local dev commands (up, test, build, clean)
  docker-compose.yml         # Local dev environment (MinIO + proxy)
  test_proxy.sh              # Bash integration tests
```

## Conventions

### Code Style

- Go 1.24+ with standard library conventions
- No external linting configuration — follow `gofmt` and `go vet` defaults
- Internal packages under `internal/` — not importable by external code
- Exported functions use GoDoc comments
- Error handling: return errors, don't panic (except `NewS3ClientFromConfig` which panics on config load failure)

### Naming

- Package names: lowercase, single word (`config`, `logger`, `s3`)
- Files: lowercase, single word (e.g., `config.go`, `s3.go`)
- Environment variables: `UPPER_SNAKE_CASE`
- Functions: `CamelCase` (exported), `camelCase` (unexported)
- Constants: defined inline or as `const` blocks

### Configuration Pattern

All configuration is via environment variables (12-factor app). The `config.FromEnv()` function parses all variables with sensible defaults. When adding new configuration:

1. Add the field to `FrontendAssetProxyConfig` struct
2. Add parsing in `FromEnv()` using `getEnv()`, `parseInt()`, or `parseDuration()` helpers
3. Document the variable in the README.md configuration table
4. Provide a reasonable default value

### Routing

Routes are defined in `cmd/proxy/main.go` using chi. The routing logic:

- `/healthz` — health check (200 OK)
- `/manifests/*` — serves from `{prefix}/{path}` (direct S3 path)
- `/apps/*` — strips `/apps`, serves from `{prefix}/data/{rest}`
- `/*` — fallback, serves from `{prefix}/data/{path}`

When adding new routes, follow the existing pattern: define the handler inline or in a dedicated function, use `s3.ProxyS3()` for S3-backed routes.

### Error Handling

S3 errors are mapped to HTTP status codes in `s3ErrorToStatus()`. The mapping covers:

- `NoSuchKey`/`NoSuchBucket` → 404
- `AccessDenied` → 403
- `context.DeadlineExceeded` → 504
- Unknown errors → 502

When handling new S3 error types, add them to `s3ErrorToStatus()`. The function unwraps `smithy.OperationError` automatically.

### Dependencies

- **AWS SDK v2** — the only significant dependency. Use `service/s3` for S3 operations
- **chi/v5** — HTTP router and middleware. Use chi's middleware stack
- **logrus** — structured logging. Use the existing `StructuredLogger` for HTTP middleware integration
- Keep dependencies minimal — this is a lightweight proxy

## Common Pitfalls

1. **SPA fallback recursion** — `ProxyS3()` has a guard against infinite recursion when the SPA entrypoint itself returns 404/403. If modifying the fallback logic, preserve this guard.
2. **S3 path resolution** — The first segment of `BUCKET_PATH_PREFIX` is treated as the bucket name. Ensure paths are correctly split when modifying `ProxyS3()`.
3. **HEAD requests** — The proxy skips body streaming for HEAD requests. When adding new response handling, check `r.Method` before writing the body.
4. **MinIO compatibility** — The S3 client uses path-style addressing (`UsePathStyle: true`) for MinIO. This is set unconditionally and works with AWS S3 too.
5. **Non-root container** — The Dockerfile runs as UID 1001. Don't add operations that require root privileges.
6. **Context timeouts** — Each S3 request gets its own timeout context (`ProxiedRequestTimeout`). Don't use the request context directly for S3 calls.
