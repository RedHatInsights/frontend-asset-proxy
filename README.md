# Caddy Frontend Asset Proxy

## Purpose

This repository contains the configuration and Docker setup for a Caddy web server designed to act as a reverse proxy for frontend assets. It is intended to be the main entrypoint for all frontend applications, redirecting requests to an S3-compatible object storage backend (like AWS S3 or Minio).

This component is part of an initiative to implement an object storage-based push cache, allowing historical and current frontend assets to be accessed and managed in an aggregated fashion.

Key functionalities include:
* Reverse proxying requests to S3/Minio.
* Supporting Single Page Application (SPA) routing by ensuring that requests for non-existent asset paths correctly serve the main application entrypoint (e.g., `index.html`). (Note: The Caddyfile in this repository is configured for this; ensure your chosen Caddyfile version supports SPA if needed).
* Providing a flexible point for potential future processing of asset requests.
* Designed to be deployed as a containerized application, managed by a Frontend Operator (FEO) within a Kubernetes environment (e.g., in the FEO namespace as a new managed resource).
* Built and versioned using Konflux.

## Features

* Based on [Caddy Web Server](https://caddyserver.com/).
* Containerized using Docker for consistent deployments.
* Configurable at runtime via environment variables.
* Includes a `/healthz` endpoint for health checks.

## Configuration (Runtime via Environment Variables)

The proxy is configured primarily through the `Caddyfile`. Runtime behavior is controlled by the following environment variables, which should be provided by the deployment environment (e.g., the Frontend Operator or Docker Compose):

| Variable                | Description                                                                    | Example (for local Minio)                    | Default (Caddyfile) | Required |
| ----------------------- | ------------------------------------------------------------------------------ | -------------------------------------------- | ------------------- | -------- |
| `SERVER_PORT`           | The internal port the Caddy server will listen on within the container.        | `8080`                                       | `8080`              | No       |
| `MINIO_UPSTREAM_URL`    | The base URL of the Minio/S3 service (scheme, host, port only).                | `http://minio:9000` (Docker Compose service name) | N/A                 | Yes      |
| `BUCKET_PATH_PREFIX`    | The bucket name/path prefix to be prepended to requests (must start with `/`). | `/frontend-assets`                           | N/A                 | Yes      |
| `SPA_ENTRYPOINT_PATH`   | Path to the SPA's entry HTML file within the bucket (e.g., `/index.html`).     | `/index.html`                                | `/index.html`       | No       |
| `LOG_LEVEL`             | The log level for Caddy (DEBUG, INFO, WARN, ERROR).                      | `DEBUG`                                      | `DEBUG` (in Caddyfile) | No       |

## Included Files

* **`Caddyfile`**: The core Caddy server configuration.
* **`Dockerfile`**: Used to build the Docker container image for this proxy.
* **`docker-compose.yml`**: For easy local setup of Caddy and Minio.
* **`Makefile`**: Provides convenient commands for common development tasks.
* **`test_caddy.sh`**: A shell script to run basic `curl` tests against a running instance of the proxy.
* **`README.md`**: This file.
* **`.gitignore`**: Specifies intentionally untracked files that Git should ignore.

## Local Setup & Testing (Using Makefile - Recommended)

The `Makefile` simplifies starting, testing, and stopping the local environment.

**Prerequisites:**
* Docker
* Docker Compose
* `make`
* Bash Shell (for `test_caddy.sh`)
* Git (to clone the repository)

**Steps:**

1.  **Clone the Repository (if you haven't already):**
    ```bash
    git clone [URL_OF_THIS_REPOSITORY]
    cd frontend-asset-proxy
    ```

2.  **Ensure Test Script is Executable (one-time setup):**
    ```bash
    chmod +x test_caddy.sh
    ```

3.  **Run Tests (This command handles setup and execution):**
    ```bash
    make test
    ```
    This command will:
    * Start Minio and Caddy services in the background using `docker-compose up -d`.
    * Prompt you to set up Minio. **This is crucial for the first run or if Minio data was cleared (`make clean-all`).**
        * Go to the Minio Console: `http://localhost:9001`
        * Log in: `minioadmin` / `minioadmin`
        * Create bucket: `frontend-assets`
        * Set `frontend-assets` bucket "Access Policy" to "Public".
        * Upload necessary test files to the `frontend-assets` bucket:
            * `index.html` (to the root of the bucket)
            * `edge-navigation.json` (to the path `api/chrome-service/v1/static/stable/prod/navigation/` within the bucket)
    * After setting up Minio, press Enter in the terminal where `make test` is waiting.
    * The `./test_caddy.sh` script will then execute.

4.  **Review Test Output:**
    The script will indicate if tests passed or failed.

5.  **View Logs (for debugging if tests fail):**
    * All services: `make logs`
    * Caddy only: `make caddy-logs`
    * Minio only: `make minio-logs`
    (Press `Ctrl+C` to stop following logs).

6.  **Stop Services:**
    When you're done:
    ```bash
    make down
    ```

**Other Useful Makefile Commands:**
* `make help`: Shows all available commands and their descriptions.
* `make up`: Starts services without running tests or prompting for Minio setup.
* `make build`: Rebuilds the Caddy Docker image (e.g., after `Caddyfile` changes).
* `make clean`: Stops and removes containers and networks.
* `make clean-all`: Stops and removes containers, networks, AND the Minio data volume (this will delete your Minio bucket and files, requiring Minio setup again).

### Manual Local Setup & Testing (Alternative)

If you prefer not to use `make` or need to perform steps individually, refer to the `docker-compose.yml` and `test_caddy.sh` script. You would typically:
1.  Start services with `docker-compose up -d`.
2.  Configure Minio as described in the `make test` step above.
3.  Run `chmod +x test_caddy.sh && ./test_caddy.sh`.
4.  Stop services with `docker-compose down`.

## Deployment

This component is designed to be deployed by the Frontend Operator (FEO). The container image will be built by Konflux and made available in the organization's container registry.

Refer to the FEO documentation for specific deployment procedures and how it manages this resource.