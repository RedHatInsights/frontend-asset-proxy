package config

import "testing"

func TestFromEnvExplicitAWSEndpoint(t *testing.T) {
	t.Setenv("MINIO_UPSTREAM_URL", "")
	t.Setenv("AWS_REGION", "eu-west-1")
	cfg := FromEnv()
	if cfg.UpstreamURL != "" || cfg.Region != "eu-west-1" {
		t.Fatalf("AWS endpoint selection lost: %+v", cfg)
	}
	t.Setenv("MINIO_UPSTREAM_URL", "https://minio.example.com:9000")
	if FromEnv().UpstreamURL != "https://minio.example.com:9000" {
		t.Fatal("custom S3 endpoint was not preserved")
	}
}
