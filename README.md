# Frontend Asset Proxy

## Purpose

This repository contains a Go reverse proxy for frontend assets. It acts as the main entrypoint for frontend applications, serving assets from an S3‑compatible object storage backend (AWS S3 or MinIO) and applying SPA‑friendly routing.

This component is part of an initiative to implement an object storage-based push cache, allowing historical and current frontend assets to be accessed and managed in an aggregated fashion.

Key functionalities include:
* Reverse proxying requests to S3/Minio.
* Supporting Single Page Application (SPA) routing by ensuring that requests for non-existent asset paths correctly serve the main application entrypoint (e.g., `index.html`).
* Providing a flexible point for potential future processing of asset requests.
* Designed to be deployed as a containerized application, managed by a Frontend Operator (FEO) within a Kubernetes environment (e.g., in the FEO namespace as a new managed resource).
* Built and versioned using Konflux.

## Features

* Go HTTP server using AWS SDK v2 (S3 GetObject streaming)
* Containerized for consistent deployments
* Configurable via environment variables
* `/healthz` endpoint for health checks

## Configuration (Environment Variables)

| Variable                | Description                                                             | Example                      | Default        |
| ----------------------- | ----------------------------------------------------------------------- | ---------------------------- | -------------- |
| `SERVER_PORT`           | Port the proxy listens on                                               | `8080`                       | `8080`         |
| `MINIO_UPSTREAM_URL`    | S3/MinIO endpoint (scheme+host[:port])                                   | `http://minio:9000`          | (empty = AWS)  |
| `BUCKET_PATH_PREFIX`    | Bucket/prefix to prepend (first segment is treated as bucket)            | `/frontend-assets`           | —              |
| `SPA_ENTRYPOINT_PATH`   | SPA entrypoint path for 403/404 fallback                                 | `/index.html`                | `/index.html`  |
| `AWS_REGION`            | AWS region (used when talking to AWS)                                     | `us-east-1`                  | `us-east-1`    |
| `AWS_ACCESS_KEY_ID`     | Access key (optional; for MinIO/local or explicit creds)                 | `minioadmin`                 | —              |
| `AWS_SECRET_ACCESS_KEY` | Secret key (optional; for MinIO/local or explicit creds)                 | `minioadmin`                 | —              |
| `AWS_CA_BUNDLE`         | (Optional) Path to custom CA bundle for TLS verification                 | `/etc/pki/ca-trust/…/ca.crt` | —              |
| `LOG_LEVEL`             | Log level for proxy and SDK (debug, info, warn, error)                   | `debug`                      | `error`        |

## Included Files

* **`cmd/proxy`**: Go entrypoint for the reverse proxy
* **`internal/s3`**: S3 client and proxy logic
* **`internal/logger`**: Structured logging and request‑scoped AWS SDK logger
* **`Dockerfile`**: Container image build
* **`docker-compose.yml`**: Local setup (MinIO + proxy)
* **`Makefile`**: Convenience commands (supports docker-compose or podman-compose)
* **`test_proxy.sh`**: Basic curl tests against the proxy

## Local Setup & Testing (Using Makefile)

The `Makefile` simplifies starting, testing, and stopping the local environment.

**Prerequisites:**
* Docker or Podman
* docker-compose or podman-compose
* `make`
* Bash Shell (for `test_proxy.sh`)
* Git (to clone the repository)

**Steps:**

1.  **Clone the Repository (if you haven't already):**
    ```bash
    git clone [URL_OF_THIS_REPOSITORY]
    cd frontend-asset-proxy
    ```

2.  **Ensure Test Script is Executable (one-time setup):**
    ```bash
    chmod +x test_proxy.sh
    ```

3.  **Run Tests (This command handles setup and execution):**
    ```bash
    make test
    ```
    This command will:
    * Start MinIO and the Go proxy in the background using `make up`.
    * Prompt you to set up Minio. **This is crucial for the first run or if Minio data was cleared (`make clean-all`).**
        * Go to the Minio Console: `http://localhost:9001`
        * Log in: `minioadmin` / `minioadmin`
        * Create bucket: `frontend-assets`
        * Set `frontend-assets` bucket "Access Policy" to "Public".
        * Upload necessary test files to the `frontend-assets` bucket:
            * `index.html` (to the root of the bucket)
            * `edge-navigation.json` (to the path `api/chrome-service/v1/static/stable/prod/navigation/` within the bucket)
    * After setting up Minio, press Enter in the terminal where `make test` is waiting.
    * The `./test_proxy.sh` script will then execute.

4.  **Review Test Output:**
    The script will indicate if tests passed or failed.

5.  **View Logs (for debugging if tests fail):**
    * All services: `make logs`
    * Proxy only: `make proxy-logs`
    * MinIO only: `make minio-logs`
    (Press `Ctrl+C` to stop following logs).

6.  **Stop Services:**
    When you're done:
    ```bash
    make down
    ```

**Other Useful Makefile Commands:**
* `make help`: Shows all available commands and their descriptions.
* `make up`: Starts services without running tests or prompting for Minio setup.
* `make build`: Rebuilds the Go proxy image.
* `make clean`: Stops and removes containers and networks.
* `make clean-all`: Stops and removes containers, networks, AND the Minio data volume (this will delete your Minio bucket and files, requiring Minio setup again).

### Manual Local Setup & Testing (Alternative)

If you prefer not to use `make` or need to perform steps individually, refer to the `docker-compose.yml` and `test_proxy.sh` script. You would typically:
1.  Start services with `docker-compose up -d` (or `podman-compose up -d`).
2.  Configure MinIO as described above.
3.  Run `chmod +x test_proxy.sh && ./test_proxy.sh`.
4.  Stop services with `docker-compose down`.

## Deployment

This component is designed to be deployed by the Frontend Operator (FEO). The container image will be built by Konflux and made available in the organization's container registry.

Refer to the FEO documentation for specific deployment procedures and how it manages this resource.
