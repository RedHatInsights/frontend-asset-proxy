package config

import (
	"os"
	"strconv"
	"time"

	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type FrontendAssetProxyConfig struct {
	// Server configuration
	ServerPort string
	LogLevel   string

	// TLS configuration
	TLSCertFile string
	TLSKeyFile  string

	// Server and proxy timeouts
	ReadHeaderTimeout     time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	IdleTimeout           time.Duration
	ProxiedRequestTimeout time.Duration
	ShutdownTimeout       time.Duration

	// Object store configuration
	UpstreamURL       string
	BucketPathPrefix  string
	SPAEntrypointPath string
	Region            string
	MaxRetryAttempts  int
	ClientLogMode     aws.ClientLogMode

	// Object store credentials
	AccessKeyID     string
	SecretAccessKey string

	// Local dev flags
	InsecureSkipVerify bool
	DisableIMDS        bool
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseInt(v string, def int) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func parseDuration(v string) time.Duration {
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0
	}
	return d
}

// parseClientLogMode parses a comma/pipe-separated list of aws log flags, or a numeric bitmask.
// Supported names (case-insensitive): retries, request, response, request_with_body, response_with_body,
// deprecated, deprecated_usage, signing, rebuilds, event_stream_body, all, none.
// If empty or invalid, defaults to aws.LogRetries|aws.LogRequest|aws.LogResponse.
func parseClientLogMode(v string) aws.ClientLogMode {
	defaultMode := aws.LogRetries | aws.LogRequest | aws.LogResponse
	if v == "" {
		return defaultMode
	}
	// Try numeric bitmask first
	if i, err := strconv.Atoi(v); err == nil {
		return aws.ClientLogMode(i)
	}

	var mode aws.ClientLogMode
	acc := strings.ReplaceAll(v, "|", ",")
	for _, p := range strings.Split(acc, ",") {
		name := strings.TrimSpace(strings.ToLower(p))
		switch name {
		case "", "none":
			// no-op
		case "retries", "retry":
			mode |= aws.LogRetries
		case "request":
			mode |= aws.LogRequest
		case "response":
			mode |= aws.LogResponse
		case "request_with_body", "requestbody", "request_body":
			mode |= aws.LogRequestWithBody
		case "response_with_body", "responsebody", "response_body":
			mode |= aws.LogResponseWithBody
		case "deprecated", "deprecated_usage":
			mode |= aws.LogDeprecatedUsage
		case "signing", "sig", "auth":
			mode |= aws.LogSigning
		case "all":
			mode |= aws.LogRetries | aws.LogRequest | aws.LogResponse | aws.LogRequestWithBody | aws.LogResponseWithBody | aws.LogDeprecatedUsage | aws.LogSigning
		}
	}
	if mode == 0 {
		return defaultMode
	}
	return mode
}

func FromEnv() FrontendAssetProxyConfig {
	cfg := FrontendAssetProxyConfig{}

	// Server configuration
	cfg.ServerPort = getEnv("SERVER_PORT", "8080")
	cfg.LogLevel = getEnv("LOG_LEVEL", "error")

	// TLS configuration
	cfg.TLSCertFile = getEnv("TLS_CERT_FILE", "")
	cfg.TLSKeyFile = getEnv("TLS_KEY_FILE", "")

	// Server and proxy timeouts with sane defaults
	cfg.ReadHeaderTimeout = parseDuration(getEnv("READ_HEADER_TIMEOUT", "5s"))
	cfg.ReadTimeout = parseDuration(getEnv("READ_TIMEOUT", "15s"))
	cfg.WriteTimeout = parseDuration(getEnv("WRITE_TIMEOUT", "60s"))
	cfg.IdleTimeout = parseDuration(getEnv("IDLE_TIMEOUT", "60s"))
	cfg.ProxiedRequestTimeout = parseDuration(getEnv("S3_GET_TIMEOUT", "60s"))
	cfg.ShutdownTimeout = parseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s"))

	// Object store configuration
	cfg.UpstreamURL = getEnv("MINIO_UPSTREAM_URL", "http://minio:9000")
	cfg.BucketPathPrefix = getEnv("BUCKET_PATH_PREFIX", "/frontend-assets")
	cfg.SPAEntrypointPath = getEnv("SPA_ENTRYPOINT_PATH", "/index.html")
	cfg.Region = getEnv("AWS_REGION", "us-east-1")
	cfg.MaxRetryAttempts = parseInt(getEnv("S3_MAX_ATTEMPTS", "3"), 3)
	cfg.ClientLogMode = parseClientLogMode(getEnv("AWS_SDK_CLIENT_LOG_MODE", ""))

	// Object store credentials
	cfg.AccessKeyID = os.Getenv("PUSHCACHE_AWS_ACCESS_KEY_ID")
	cfg.SecretAccessKey = os.Getenv("PUSHCACHE_AWS_SECRET_ACCESS_KEY")

	return cfg
}
