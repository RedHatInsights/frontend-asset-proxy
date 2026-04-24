@AGENTS.md

## Build Commands

```bash
# Build the binary locally
CGO_ENABLED=0 go build -o frontend-asset-proxy cmd/proxy/main.go

# Build the Docker image
make build

# Start local dev environment (MinIO + proxy)
make up

# Run integration tests (starts services if needed)
make test

# Stop services
make down

# Clean everything including MinIO data
make clean-all
```

## Verification

After making changes, always run:

```bash
# Verify Go compilation
go build ./...

# Run vet checks
go vet ./...

# Run unit tests (if any exist)
go test ./...

# Verify Docker build
make build
```

## CI Checks

PRs trigger Konflux/Tekton pipelines (`.tekton/`) that build the container image. The pipeline does not run tests — only validates the Docker build succeeds.
