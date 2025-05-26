# Caddy Frontend Asset Proxy

## Purpose

This repository contains the configuration and Docker setup for a Caddy web server designed to act as a reverse proxy for frontend assets. It is intended to be the main entrypoint for all frontend applications, redirecting requests to an S3-compatible object storage backend (like AWS S3 or Minio).

This component is part of an initiative to implement an object storage-based push cache, allowing historical and current frontend assets to be accessed and managed in an aggregated fashion.

Key functionalities include:
* Reverse proxying requests to S3/Minio.
* Supporting Single Page Application (SPA) routing by ensuring that requests for non-existent asset paths correctly serve the main application entrypoint (e.g., `index.html`).
* Providing a flexible point for potential future processing of asset requests.
* Designed to be deployed as a containerized application, managed by a Frontend Operator (FEO) within a Kubernetes environment (e.g., in the FEO namespace as a new managed resource).
* Built and versioned using Konflux.

## Features

* Based on [Caddy Web Server](https://caddyserver.com/).
* Containerized using Docker for consistent deployments.
* Configurable at runtime via environment variables.
* Includes SPA fallback routing.
* Provides a `/healthz` endpoint for health checks.

## Configuration (Runtime via Environment Variables)

The proxy is configured primarily through the `Caddyfile` located in the root of this repository. Runtime behavior is controlled by the following environment variables, which should be provided by the deployment environment (e.g., the Frontend Operator):

| Variable                | Description                                                                    | Example (for local Minio)                    | Default (if any) | Required |
| ----------------------- | ------------------------------------------------------------------------------ | -------------------------------------------- | ---------------- | -------- |
| `SERVER_PORT`           | The internal port the Caddy server will listen on within the container.        | `8080`                                       | `8080`           | No       |
| `ASSET_BACKEND_URL`     | The URL of the S3/Minio bucket root where assets are stored.                     | `http://minio-dev:9000/frontend-assets`      | N/A              | Yes      |
| `SPA_ENTRYPOINT_PATH`   | The path to the SPA's entry HTML file (e.g., index.html) within the asset backend. | `/index.html`                                | `/index.html`    | No       |
| `LOG_LEVEL`             | The log level for Caddy (DEBUG, INFO, WARN, ERROR).                      | `INFO`                                       | `INFO`           | No       |

## Included Files

* **`Caddyfile`**: The core Caddy server configuration.
* **`Dockerfile`**: Used to build the Docker container image for this proxy.
* **`test_caddy.sh`**: A shell script to run basic `curl` tests against a running instance of the proxy.
* **`README.md`**: This file.
* **`.gitignore`**: Specifies intentionally untracked files that Git should ignore.

## Local Setup & Testing

These instructions guide you through setting up and testing the Caddy proxy locally on a Linux Fedora system (adaptable for other OS with Docker).

### Prerequisites

1.  **Docker:** Ensure Docker is installed and running.
2.  **Git:** To clone this repository.
3.  **Text Editor:** For viewing/editing files.
4.  **Bash Shell:** For running the test script.

### Steps

1.  **Clone the Repository:**
    ```bash
    git clone [URL_OF_THIS_REPOSITORY]
    cd frontend-asset-proxy
    ```

2.  **Set Up Minio (Local S3-Compatible Backend):**
    * Create a Docker network (if it doesn't exist):
        ```bash
        docker network create my-dev-network
        ```
    * Create a local directory for Minio data:
        ```bash
        mkdir -p ~/minio/data
        ```
    * Run the Minio container:
        ```bash
        docker run -d \
          --name minio-dev \
          --network my-dev-network \
          -p 9000:9000 \
          -p 9001:9001 \
          -v ~/minio/data:/data:Z \
          -e "MINIO_ROOT_USER=minioadmin" \
          -e "MINIO_ROOT_PASSWORD=minioadmin" \
          minio/minio server /data --console-address ":9001"
        ```
        *(Note the `:Z` on the volume mount for SELinux on Fedora).*
    * **Configure Minio:**
        * Open your browser to `http://localhost:9001`.
        * Log in with `minioadmin` / `minioadmin`.
        * Create a bucket named `frontend-assets`.
        * Set the `frontend-assets` bucket's "Access Policy" to "Public".
        * Upload a sample `index.html` to the root of the `frontend-assets` bucket. You can use the `index.html` provided in previous examples or create your own.
        * (Optional) Upload a `css/style.css` file if you want to test the asset path in `test_caddy.sh` directly.

3.  **Build the Caddy Proxy Docker Image:**
    Navigate to the root of this repository (where the `Dockerfile` is located) and run:
    ```bash
    docker build -t frontend-asset-proxy:local .
    ```

4.  **Run the Caddy Proxy Container:**
    ```bash
    docker run --rm -p 8080:8080 --name my-caddy-proxy \
      --network my-dev-network \
      -e SERVER_PORT="8080" \
      -e ASSET_BACKEND_URL="http://minio-dev:9000/frontend-assets" \
      -e SPA_ENTRYPOINT_PATH="/index.html" \
      -e LOG_LEVEL="DEBUG" \
      frontend-asset-proxy:local
    ```
    * `--rm`: Automatically removes the container when it exits.
    * You should see Caddy startup logs in your terminal.

5.  **Test the Caddy Proxy:**
    * **Manual Browser Test:**
        * Open `http://localhost:8080/` in your browser. You should see your `index.html`.
        * Try `http://localhost:8080/some/non-existent/spa-route`. It should also serve `index.html`.
    * **Using the Test Script:**
        * Open a new terminal window.
        * Navigate to the repository directory.
        * Make the script executable: `chmod +x test_caddy.sh`
        * Run the script: `./test_caddy.sh`
        * Review the output for test successes or failures.

6.  **View Caddy Logs:**
    If you ran the Caddy container in detached mode (`-d`), you can view its logs with:
    ```bash
    docker logs my-caddy-proxy
    ```

7.  **Stopping Local Services:**
    * To stop Caddy: Press `Ctrl+C` in the terminal where it's running (if attached), or `docker stop my-caddy-proxy` (if detached).
    * To stop Minio: `docker stop minio-dev`