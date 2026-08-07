// Package domain contains the shared types of the outbox demo: the business
// entity (Order), the event that travels through the outbox (Event) and the
// outbox row as stored in the database (OutboxRow).
package domain

import (
	"encoding/json"
	"time"
)

// EventPaymentAuthorized is the event type emitted when an order is created.
const EventPaymentAuthorized = "PaymentAuthorized"

// Order is the business state: a registered payment/order.
type Order struct {
	ID        string    `json:"id"`
	Customer  string    `json:"customer"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Event is the payload stored in the outbox and delivered to the broker.
type Event struct {
	Type     string  `json:"type"`
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// OutboxRow mirrors a row of the outbox table.
type OutboxRow struct {
	ID          int64           `json:"id"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	PublishedAt *time.Time      `json:"published_at"`
}
