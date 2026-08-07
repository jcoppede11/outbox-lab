// Command server wires together the store, the fake broker, the relay goroutine
// and the HTTP API for the outbox demo.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"outbox-lab/internal/api"
	"outbox-lab/internal/broker"
	"outbox-lab/internal/relay"
	"outbox-lab/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.databaseURL)
	if err != nil {
		log.Error("connect to postgres failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Error("ping postgres failed", "err", err)
		os.Exit(1)
	}
	log.Info("connected to postgres")

	st := store.New(pool)
	bk := broker.New()

	rl := relay.New(st, bk.Publish, cfg.relayInterval, cfg.relayBatch, log)
	go rl.Run(ctx)

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           api.New(st, bk, log).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("http server listening", "addr", cfg.addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
	}
}

type config struct {
	databaseURL   string
	addr          string
	relayInterval time.Duration
	relayBatch    int
}

func loadConfig() config {
	return config{
		databaseURL:   env("DATABASE_URL", "postgres://outbox:outbox@localhost:5432/outbox_lab?sslmode=disable"),
		addr:          env("HTTP_ADDR", ":8080"),
		relayInterval: envDuration("RELAY_INTERVAL", time.Second),
		relayBatch:    envInt("RELAY_BATCH", 50),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
