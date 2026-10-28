// Command api runs the FrameCheck HTTP API and its sequential analysis worker.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"framecheck/internal/adapters/fixture"
	httpadapter "framecheck/internal/adapters/http"
	"framecheck/internal/adapters/meta"
	"framecheck/internal/adapters/mock"
	"framecheck/internal/adapters/postgres"
	"framecheck/internal/adapters/storage"
	"framecheck/internal/application"
	"framecheck/internal/ports"
)

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type uuidGen struct{}

func (uuidGen) NewID() string { return uuid.NewString() }

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	loadDotEnv(log, ".env", filepath.Join("..", ".env"))

	dbURL := os.Getenv("FRAMECHECK_DATABASE_URL")
	if dbURL == "" {
		return errors.New("FRAMECHECK_DATABASE_URL is not set")
	}
	provider, err := selectProvider(log, envOr("FRAMECHECK_PROVIDER", "mock"))
	if err != nil {
		return err
	}
	addr := "127.0.0.1:" + envOr("FRAMECHECK_API_PORT", "8089")
	storageDir := envOr("FRAMECHECK_STORAGE_DIR", "../storage")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(pingCtx)
	cancel()
	if err != nil {
		return errors.New("cannot reach PostgreSQL; is the framecheck compose project running?")
	}
	if err := postgres.Migrate(ctx, pool); err != nil {
		return err
	}

	store, err := storage.NewLocal(storageDir)
	if err != nil {
		return err
	}
	svc := application.NewService(application.Deps{
		Repo: postgres.NewRepository(pool), Store: store, Provider: provider,
		Clock: systemClock{}, IDs: uuidGen{}, Log: log,
	})
	if err := svc.RecoverInterrupted(ctx); err != nil {
		return err
	}
	go svc.Work(ctx)

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpadapter.NewHandler(svc, log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Info("framecheck api listening", "addr", addr, "provider", provider.Origin(), "model", provider.ModelID())

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// selectProvider picks the analysis provider. Only "meta" makes network
// calls, and only when MODEL_API_ENABLED=true is also set, because every
// call may be billed.
func selectProvider(log *slog.Logger, name string) (ports.InferenceProvider, error) {
	enabled := strings.EqualFold(os.Getenv("MODEL_API_ENABLED"), "true")
	switch name {
	case "mock":
		return mock.New(), nil
	case "fixture":
		return fixture.Load(strings.Split(envOr("FRAMECHECK_FIXTURES_DIR", "../demo/fixtures,../demo/fixtures-holdout"), ",")...)
	case "meta":
		if !enabled {
			return nil, errors.New("FRAMECHECK_PROVIDER=meta needs MODEL_API_ENABLED=true; set it only after credits and rules are confirmed")
		}
		maxReq, err1 := strconv.ParseInt(envOr("MODEL_API_MAX_REQUESTS", "20"), 10, 64)
		maxSide, err2 := strconv.Atoi(envOr("MODEL_API_MAX_IMAGE_SIDE", "2048"))
		maxOut, err3 := strconv.Atoi(envOr("MODEL_API_MAX_OUTPUT_TOKENS", "4000"))
		if err := errors.Join(err1, err2, err3); err != nil || maxReq < 1 {
			return nil, errors.New("MODEL_API_MAX_REQUESTS, MODEL_API_MAX_IMAGE_SIDE, and MODEL_API_MAX_OUTPUT_TOKENS must be positive integers")
		}
		p, err := meta.New(meta.Config{
			BaseURL:         envOr("MODEL_API_BASE_URL", "https://api.meta.ai/v1"),
			APIKey:          os.Getenv("MODEL_API_KEY"),
			Model:           envOr("MODEL_API_SPARK_MODEL", "muse-spark-1.3"),
			MaxOutputTokens: maxOut,
			MaxImageSide:    maxSide,
			MaxRequests:     maxReq,
		})
		if err != nil {
			return nil, err
		}
		log.Warn("Meta Model API enabled: every run is a billable request", "model", p.ModelID(), "max_requests_this_process", maxReq)
		return p, nil
	}
	return nil, fmt.Errorf("FRAMECHECK_PROVIDER must be mock, fixture, or meta, got %q", name)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadDotEnv reads the first existing KEY=VALUE file without overriding
// variables already set in the environment. Values are never logged.
func loadDotEnv(log *slog.Logger, paths ...string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			if _, set := os.LookupEnv(k); !set {
				os.Setenv(k, strings.Trim(strings.TrimSpace(v), `"'`))
			}
		}
		log.Info("loaded environment file", "path", p)
		return
	}
}
