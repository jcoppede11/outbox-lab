// Package store holds all database access for the outbox demo. It is the only
// place that talks to PostgreSQL, and it is where the two invariants of the
// pattern live: (1) a payment and its event are written in one transaction, and
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

// CreatePayment inserts the payment and its outbox event inside a single
// transaction. Either both rows commit or neither does — this atomicity is the
// whole point of the outbox pattern: we never charge without recording the
// event, nor emit an event without a real charge.
func (s *Store) CreatePayment(ctx context.Context, customer string, amount float64, currency string) (domain.Payment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Payment{}, err
	}
	defer tx.Rollback(ctx) // no-op once the tx is committed

	var p domain.Payment
	err = tx.QueryRow(ctx,
		`INSERT INTO payments (customer, amount, currency, status)
		 VALUES ($1, $2, $3, 'authorized')
		 RETURNING id, customer, amount::float8, currency, status, created_at`,
		customer, amount, currency,
	).Scan(&p.ID, &p.Customer, &p.Amount, &p.Currency, &p.Status, &p.CreatedAt)
	if err != nil {
		return domain.Payment{}, err
	}

	payload, err := json.Marshal(domain.Event{
		Type:      domain.EventPaymentAuthorized,
		PaymentID: p.ID,
		Amount:    p.Amount,
		Currency:  p.Currency,
	})
	if err != nil {
		return domain.Payment{}, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO outbox (payload) VALUES ($1)`, payload,
	); err != nil {
		return domain.Payment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Payment{}, err
	}
	return p, nil
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

// Payments returns all payments, newest first.
func (s *Store) Payments(ctx context.Context) ([]domain.Payment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, customer, amount::float8, currency, status, created_at
		 FROM payments ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make([]domain.Payment, 0)
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.Customer, &p.Amount, &p.Currency, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
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
