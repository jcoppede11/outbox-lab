// Package broker is a fake, in-process message broker. It stands in for what
// billing and notifications actually receive. It can be "torn down" to simulate
// an outage: while down, publishing fails and the relay must retry — which is
// exactly what demonstrates the at-least-once guarantee of the outbox pattern.
package broker

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"outbox-lab/internal/domain"
)

// ErrDown is returned by Publish while the broker is torn down.
var ErrDown = errors.New("broker is down")

// Broker records every event it receives. It is safe for concurrent use.
type Broker struct {
	mu       sync.Mutex
	down     bool
	received []domain.Event
}

// New returns an up (reachable) broker.
func New() *Broker { return &Broker{} }

// Publish delivers an event to the broker. Returns ErrDown when torn down.
func (b *Broker) Publish(_ context.Context, payload json.RawMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.down {
		return ErrDown
	}
	var ev domain.Event
	if err := json.Unmarshal(payload, &ev); err != nil {
		return err
	}
	b.received = append(b.received, ev)
	return nil
}

// SetDown tears the broker down (true) or brings it back up (false).
func (b *Broker) SetDown(down bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.down = down
}

// IsDown reports whether the broker is currently torn down.
func (b *Broker) IsDown() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.down
}

// Received returns a copy of every event delivered so far.
func (b *Broker) Received() []domain.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]domain.Event, len(b.received))
	copy(out, b.received)
	return out
}
