// Package store holds all database access for the outbox demo. It is the only
// place that talks to PostgreSQL, and it is where the two invariants of the
// pattern live: (1) an order and its event are written in one transaction, and
// (2) the relay drains pending events transactionally.
package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"outbox-lab/internal/domain"
)

// Store is a thin data-access layer over a pgx connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store backed by the given pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// CreateOrder inserts the order and its outbox event inside a single
// transaction. Either both rows commit or neither does — this atomicity is the
// whole point of the outbox pattern: we never charge without recording the
// event, nor emit an event without a real charge.
func (s *Store) CreateOrder(ctx context.Context, customer string, amount float64, currency string) (domain.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Order{}, err
	}
	defer tx.Rollback(ctx) // no-op once the tx is committed

	var o domain.Order
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (customer, amount, currency, status)
		 VALUES ($1, $2, $3, 'authorized')
		 RETURNING id, customer, amount::float8, currency, status, created_at`,
		customer, amount, currency,
	).Scan(&o.ID, &o.Customer, &o.Amount, &o.Currency, &o.Status, &o.CreatedAt)
	if err != nil {
		return domain.Order{}, err
	}

	payload, err := json.Marshal(domain.Event{
		Type:     domain.EventPaymentAuthorized,
		OrderID:  o.ID,
		Amount:   o.Amount,
		Currency: o.Currency,
	})
	if err != nil {
		return domain.Order{}, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO outbox (payload) VALUES ($1)`, payload,
	); err != nil {
		return domain.Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, err
	}
	return o, nil
}

// PublishFunc publishes one event payload to the broker. It returns an error
// when the broker is unreachable, in which case the caller keeps the row
// pending so it is retried later.
type PublishFunc func(ctx context.Context, payload json.RawMessage) error

// DrainPending fetches up to `limit` pending events (ordered, FIFO), publishes
// each and marks it as sent within the same transaction. FOR UPDATE SKIP LOCKED
// means several relay instances could run concurrently without stepping on each
// other. When publishing fails the loop stops and the transaction commits only
// the events published so far; the rest stay pending and are retried on the
// next tick. Returns how many events were published.
func (s *Store) DrainPending(ctx context.Context, publish PublishFunc, limit int) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx,
		`SELECT id, payload FROM outbox
		 WHERE status = 'pending'
		 ORDER BY id
		 FOR UPDATE SKIP LOCKED
		 LIMIT $1`, limit)
	if err != nil {
		return 0, err
	}

	type pending struct {
		id      int64
		payload json.RawMessage
	}
	var batch []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.payload); err != nil {
			rows.Close()
			return 0, err
		}
		batch = append(batch, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	sent := 0
	for _, p := range batch {
		if err := publish(ctx, p.payload); err != nil {
			// Broker down or transient failure: stop here. Rows not yet marked
			// stay pending; already-marked ones are committed below.
			break
		}
		if _, err := tx.Exec(ctx,
			`UPDATE outbox SET status = 'sent', published_at = now() WHERE id = $1`, p.id,
		); err != nil {
			return 0, err
		}
		sent++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return sent, nil
}

// Orders returns all orders, newest first.
func (s *Store) Orders(ctx context.Context) ([]domain.Order, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, customer, amount::float8, currency, status, created_at
		 FROM orders ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.Customer, &o.Amount, &o.Currency, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// Outbox returns all outbox rows, newest first.
func (s *Store) Outbox(ctx context.Context) ([]domain.OutboxRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, payload, status, created_at, published_at
		 FROM outbox ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.OutboxRow, 0)
	for rows.Next() {
		var r domain.OutboxRow
		if err := rows.Scan(&r.ID, &r.Payload, &r.Status, &r.CreatedAt, &r.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
