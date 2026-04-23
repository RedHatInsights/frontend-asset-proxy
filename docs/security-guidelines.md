# Security Guidelines

Security considerations for the frontend-asset-proxy service.

## Credential Handling

### S3 Credentials

Access keys are passed via environment variables (`PUSHCACHE_AWS_ACCESS_KEY_ID`, `PUSHCACHE_AWS_SECRET_ACCESS_KEY`). These must **never** be:

- Hardcoded in source code or configuration files
- Logged to stdout/stderr (even in debug mode)
- Included in error messages or stack traces
- Committed to the repository in any form (including `.env` files)

### Credential Provider Chain

The S3 client in `internal/s3/s3.go` uses a priority-ordered credential chain:

1. **Explicit credentials** — `PUSHCACHE_AWS_ACCESS_KEY_ID` + `PUSHCACHE_AWS_SECRET_ACCESS_KEY` (for local/MinIO)
2. **IMDS (EC2 Instance Metadata)** — used in AWS deployments with IAM roles
3. **Anonymous credentials** — fallback when no other provider matches

When adding new credential sources, maintain this priority order. IMDS can be disabled via `DISABLE_IMDS=true` for non-AWS environments.

### Local Development Credentials

The `docker-compose.yml` uses `minioadmin:minioadmin` for local MinIO. These are test-only values and must never appear in production configuration. The `.dockerignore` already excludes `.docker`, `.kube`, and `.podman` directories to prevent secrets from leaking into container images.

## Container Security

- The container runs as **non-root** (UID 1001) — do not change this
- The binary is statically linked (`CGO_ENABLED=0`) — no shared library dependencies
- Base image is **UBI9-minimal** — minimal attack surface
- Only ports 8080 (HTTP) is exposed

## TLS Configuration

TLS is optional and configured via `TLS_CERT_FILE` and `TLS_KEY_FILE` environment variables. When both are set, the server uses `ListenAndServeTLS`. When adding TLS-related code:

- Never disable certificate verification in production
- The `InsecureSkipVerify` config option exists for local development with self-signed certificates only
- Custom CA bundles are supported via `AWS_CA_BUNDLE` for S3 connections

## Request Handling

### Path Traversal

S3 key resolution uses `url.PathUnescape()` on the request path. The S3 API provides its own path validation, but when adding new route handlers:

- Never construct file system paths from user input
- Always validate that resolved S3 keys stay within the expected bucket prefix
- The `BUCKET_PATH_PREFIX` config defines the allowed scope

### HTTP Methods

Only `GET` and `HEAD` methods are allowed. The proxy returns `405 Method Not Allowed` for all others. Do not add write methods (`PUT`, `POST`, `DELETE`) without explicit security review — this proxy is read-only by design.

### Error Information

S3 errors are mapped to HTTP status codes in `s3ErrorToStatus()`. Error responses must not expose:

- Internal S3 bucket names or paths
- AWS SDK error details or stack traces
- Credential or configuration information

The current implementation returns only HTTP status codes without response bodies for error cases.
