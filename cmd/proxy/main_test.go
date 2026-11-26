package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/frontend-asset-proxy/internal/config"
	"github.com/RedHatInsights/frontend-asset-proxy/internal/logger"
	"github.com/sirupsen/logrus"
)

func TestLocaleConfigRoutes(t *testing.T) {
	objects := map[string]string{
		"/assets/data/config/locales.json":     `[{"id":"fr","file":"messages.fr.json"}]`,
		"/assets/data/config/messages.fr.json": `{"bundle.iam.title":"Identité"}`,
		"/assets/index.html":                   "<html>SPA</html>",
	}
	var requested []string
	store := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		if body, ok := objects[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=60")
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			_, _ = io.WriteString(w, body)
			return
		}
		status, code := http.StatusNotFound, "NoSuchKey"
		if strings.HasSuffix(r.URL.Path, "messages.denied.json") {
			status, code = http.StatusForbidden, "AccessDenied"
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		_, _ = fmt.Fprintf(w, "<Error><Code>%s</Code></Error>", code)
	}))
	defer store.Close()
	cfg := config.FrontendAssetProxyConfig{
		UpstreamURL: store.URL, BucketPathPrefix: "/assets", Region: "us-east-1",
		SPAEntrypointPath: "/index.html", ProxiedRequestTimeout: time.Second,
		MaxRetryAttempts: 1, AccessKeyID: "test", SecretAccessKey: "test",
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	router := newRouter(cfg, logger.NewLogger("error", log))
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		for _, prefix := range []string{"/config/", "/apps/config/"} {
			for _, file := range []string{"locales.json", "messages.fr.json", "messages.missing.json", "messages.denied.json"} {
				t.Run(method+prefix+file, func(t *testing.T) {
					requested = nil
					response := httptest.NewRecorder()
					router.ServeHTTP(response, httptest.NewRequest(method, prefix+file, nil))
					want := http.StatusOK
					if strings.Contains(file, "missing") {
						want = http.StatusNotFound
					}
					if strings.Contains(file, "denied") {
						want = http.StatusForbidden
					}
					if response.Code != want {
						t.Fatalf("status %d, want %d: %s", response.Code, want, response.Body.String())
					}
					if len(requested) != 1 || requested[0] != "/assets/data/config/"+file {
						t.Fatalf("unexpected storage requests (must not use SPA fallback): %v", requested)
					}
					if want == http.StatusOK {
						if response.Header().Get("Content-Type") != "application/json" || response.Header().Get("Cache-Control") != "public, max-age=60" {
							t.Fatalf("lost object headers: %v", response.Header())
						}
						if method == http.MethodGet && response.Body.String() != objects[requested[0]] {
							t.Fatal("changed locale bytes")
						}
						if method == http.MethodHead && response.Body.Len() != 0 {
							t.Fatal("HEAD returned a body")
						}
					}
				})
			}
		}
	}
	// A client refresh receives the new object, without restarting the proxy.
	objects["/assets/data/config/messages.fr.json"] = `{"bundle.iam.title":"Comptes"}`
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/config/messages.fr.json", nil))
	if response.Code != 200 || response.Body.String() != objects["/assets/data/config/messages.fr.json"] {
		t.Fatal("proxy retained stale locale content")
	}
	// Ordinary SPA paths retain their existing fallback.
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/settings/users", nil))
	if response.Code != 200 || response.Body.String() != "<html>SPA</html>" {
		t.Fatalf("broke SPA fallback: %d %s", response.Code, response.Body.String())
	}
}
