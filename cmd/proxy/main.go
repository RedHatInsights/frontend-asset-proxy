package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/RedHatInsights/frontend-asset-proxy/internal/config"
	"github.com/RedHatInsights/frontend-asset-proxy/internal/logger"
	"github.com/RedHatInsights/frontend-asset-proxy/internal/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg := config.FromEnv()
	listen := cfg.ServerPort
	upstream := cfg.UpstreamURL
	prefix := cfg.BucketPathPrefix
	level := cfg.LogLevel
	structuredLogger := logger.NewLogger(level, logrus.New())
	log := structuredLogger.Logger

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(structuredLogger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)

	s3Client := s3.NewS3ClientFromConfig(cfg, log)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// /manifests/* -> /{prefix}{original}
	r.Get("/manifests/*", func(w http.ResponseWriter, r *http.Request) {
		full := s3.JoinPath(prefix, r.URL.Path)
		s3.ProxyS3(w, r, s3Client, cfg, full, log)
	})

	// /apps/* -> /{prefix}/data/{rest}
	r.Get("/apps/*", func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, "/apps")
		full := s3.JoinPath(prefix, "/data"+trimmed)
		s3.ProxyS3(w, r, s3Client, cfg, full, log)
	})

	// handle HEAD requests
	r.MethodFunc(http.MethodHead, "/*", func(w http.ResponseWriter, r *http.Request) {
		full := s3.JoinPath(prefix, "/data"+r.URL.Path)
		s3.ProxyS3(w, r, s3Client, cfg, full, log)
	})

	// fallback: prepend {prefix}/data
	r.MethodFunc(http.MethodGet, "/*", func(w http.ResponseWriter, r *http.Request) {
		full := s3.JoinPath(prefix, "/data"+r.URL.Path)
		s3.ProxyS3(w, r, s3Client, cfg, full, log)
	})

	// Return 405 for unsupported methods on matched routes
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})

	srv := &http.Server{
		Addr:              ":" + listen,
		Handler:           r,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	certFile := cfg.TLSCertFile
	keyFile := cfg.TLSKeyFile
	log.Printf("proxy listening on :%s (tls=%v) -> %s (prefix=%s)", listen, certFile != "" && keyFile != "", upstream, prefix)
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)

	go func() {
		var err error
		if certFile != "" && keyFile != "" {
			err = srv.ListenAndServeTLS(certFile, keyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-interrupts
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
