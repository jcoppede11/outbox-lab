// Package api exposes the HTTP surface of the demo: the transactional writer
// (POST /api/payments), a read endpoint that shows all three actors at once
// (GET /api/state) and the chaos switch for the broker (POST /api/chaos).
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"outbox-lab/internal/broker"
	"outbox-lab/internal/domain"
	"outbox-lab/internal/store"
)

// Server wires the HTTP handlers to the store and the broker.
type Server struct {
	store  *store.Store
	broker *broker.Broker
	log    *slog.Logger
}

// New builds a Server.
func New(s *store.Store, b *broker.Broker, log *slog.Logger) *Server {
	return &Server{store: s, broker: b, log: log}
}

// Routes returns the HTTP handler with all routes and CORS applied.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/payments", s.createPayment)
	mux.HandleFunc("GET /api/state", s.getState)
	mux.HandleFunc("POST /api/chaos", s.setChaos)
	return cors(mux)
}

type createPaymentRequest struct {
	Customer string  `json:"customer"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// createPayment registers a payment and its outbox event atomically.
func (s *Server) createPayment(w http.ResponseWriter, r *http.Request) {
	var req createPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Customer == "" || req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "customer is required and amount must be > 0")
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	payment, err := s.store.CreatePayment(r.Context(), req.Customer, req.Amount, req.Currency)
	if err != nil {
		s.log.Error("create payment failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create payment")
		return
	}
	writeJSON(w, http.StatusCreated, payment)
}

type stateResponse struct {
	Payments []domain.Payment   `json:"payments"`
	Outbox   []domain.OutboxRow `json:"outbox"`
	Broker   brokerState        `json:"broker"`
}

type brokerState struct {
	Down          bool           `json:"down"`
	ReceivedCount int            `json:"received_count"`
	Received      []domain.Event `json:"received"`
}

// getState returns the three visible actors: payments, outbox and broker.
func (s *Server) getState(w http.ResponseWriter, r *http.Request) {
	payments, err := s.store.Payments(r.Context())
	if err != nil {
		s.log.Error("list payments failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not read payments")
		return
	}
	outbox, err := s.store.Outbox(r.Context())
	if err != nil {
		s.log.Error("list outbox failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not read outbox")
		return
	}
	received := s.broker.Received()
	writeJSON(w, http.StatusOK, stateResponse{
		Payments: payments,
		Outbox:   outbox,
		Broker: brokerState{
			Down:          s.broker.IsDown(),
			ReceivedCount: len(received),
			Received:      received,
		},
	})
}

type chaosRequest struct {
	Down bool `json:"down"`
}

// setChaos tears the broker down or brings it back up.
func (s *Server) setChaos(w http.ResponseWriter, r *http.Request) {
	var req chaosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s.broker.SetDown(req.Down)
	s.log.Info("chaos toggled", "broker_down", req.Down)
	writeJSON(w, http.StatusOK, chaosRequest{Down: s.broker.IsDown()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// cors allows the Angular dev server to call the API from another origin.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
