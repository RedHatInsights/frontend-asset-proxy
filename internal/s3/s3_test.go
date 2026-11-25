package s3

import (
	"context"
	"io"
	"testing"

	"github.com/RedHatInsights/frontend-asset-proxy/internal/config"
	"github.com/sirupsen/logrus"
)

func TestS3ClientAWSAndCustomEndpoints(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	for _, endpoint := range []string{"", "https://s3.eu-west-1.amazonaws.com:443", "http://localhost:9000"} {
		t.Run(endpoint, func(t *testing.T) {
			cfg := config.FrontendAssetProxyConfig{UpstreamURL: endpoint, Region: "eu-west-1", AccessKeyID: "test", SecretAccessKey: "test"}
			options := NewS3ClientFromConfig(cfg, log).Options()
			if options.Region != cfg.Region {
				t.Fatal("region lost")
			}
			if endpoint == "" {
				if options.BaseEndpoint != nil || options.UsePathStyle {
					t.Fatal("AWS SDK regional endpoint resolution overridden")
				}
			} else if options.BaseEndpoint == nil || *options.BaseEndpoint != endpoint || !options.UsePathStyle {
				t.Fatalf("custom endpoint not configured: %+v", options)
			}
			credentials, err := options.Credentials.Retrieve(context.Background())
			if err != nil || credentials.AccessKeyID != "test" || credentials.SecretAccessKey != "test" {
				t.Fatal("configured credential provider not preserved")
			}
		})
	}
}
