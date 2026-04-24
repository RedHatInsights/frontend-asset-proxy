# Testing Guidelines

Testing patterns and practices for the frontend-asset-proxy service.

## Current Test Infrastructure

### Integration Tests

The primary test suite is `test_proxy.sh` — a bash script that runs curl-based HTTP tests against the running proxy. It validates:

- Health endpoint (`/healthz`) — status code, content type, body
- Asset serving via `/apps/*` route — HTML and JS files
- Manifest serving via `/manifests/*` route
- Response headers (Content-Type, status codes)
- Response body content snippets

### Running Tests

```bash
# Full test flow: start services, prompt for MinIO setup, run tests
make test

# Run tests manually (services must be running)
./test_proxy.sh
```

### Test Environment

Tests require `docker-compose` (or `podman-compose`) with:

- **MinIO** container on port 9000 (S3 API) and 9001 (console)
- **Proxy** container on port 8080
- A `frontend-assets` bucket in MinIO with test files uploaded

First-run setup requires manual MinIO configuration via the web console at `http://localhost:9001` (credentials: `minioadmin`/`minioadmin`).

## Go Unit Tests

There are currently no Go unit tests (`*_test.go` files) in the repository. When adding unit tests:

### Conventions

- Place test files alongside source files (`internal/s3/s3_test.go`)
- Use the standard `testing` package — no external test frameworks
- Use table-driven tests for functions with multiple input/output combinations
- Test file names: `<source>_test.go`
- Test function names: `Test<FunctionName>_<scenario>`

### What to Test

Priority areas for unit test coverage:

1. **`internal/config/config.go`** — environment variable parsing, defaults, edge cases (invalid values, missing vars)
2. **`internal/s3/s3.go`** — `s3ErrorToStatus()` error mapping, `JoinPath()` path joining, bucket/key parsing from paths
3. **`internal/logger/logger.go`** — log level parsing, classification mapping

### Running Go Tests

```bash
go test ./...              # Run all tests
go test ./internal/s3/     # Run tests for a specific package
go test -v -run TestName   # Run a specific test with verbose output
```

## Adding New Test Files

When adding test assets for integration tests:

1. Update `test_proxy.sh` with new test cases
2. Document any new files that need to be uploaded to MinIO in the test setup instructions
3. Use descriptive test names and clear pass/fail output
4. Return non-zero exit code on any failure

## CI/CD Testing

The Tekton/Konflux pipeline (`.tekton/`) builds the container image but does not currently run tests. The pipeline uses `docker-build-oci-ta` which only builds the Dockerfile. Any test execution happens locally via `make test`.
