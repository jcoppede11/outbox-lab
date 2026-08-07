// Package relay contains the background process that drains the outbox. It runs
// as a goroutine driven by a time.Ticker: on every tick it asks the store to
// publish pending events to the broker and mark them as sent.
package relay

import (
	"context"
	"log/slog"
	"time"

	"outbox-lab/internal/store"
)

// Drainer is the slice of the store the relay depends on.
type Drainer interface {
	DrainPending(ctx context.Context, publish store.PublishFunc, limit int) (int, error)
}

// Relay periodically drains pending outbox events to the broker.
type Relay struct {
	store    Drainer
	publish  store.PublishFunc
	interval time.Duration
	batch    int
	log      *slog.Logger
}

// New builds a Relay. publish is the broker's Publish method.
func New(d Drainer, publish store.PublishFunc, interval time.Duration, batch int, log *slog.Logger) *Relay {
	return &Relay{store: d, publish: publish, interval: interval, batch: batch, log: log}
}

// Run polls the outbox on the ticker until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.log.Info("relay started", "interval", r.interval, "batch", r.batch)
	for {
		select {
		case <-ctx.Done():
			r.log.Info("relay stopped")
			return
		case <-ticker.C:
			n, err := r.store.DrainPending(ctx, r.publish, r.batch)
			if err != nil {
				r.log.Error("relay drain failed", "err", err)
				continue
			}
			if n > 0 {
				r.log.Info("relay published events", "count", n)
			}
		}
	}
}
