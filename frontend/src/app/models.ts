/** Tipos que reflejan las respuestas del backend Go (GET /api/state). */

export type OutboxStatus = 'pending' | 'sent' | 'failed';

export interface Payment {
  id: string;
  customer: string;
  amount: number;
  currency: string;
  status: string;
  created_at: string;
}

export interface PaymentEvent {
  type: string;
  payment_id: string;
  amount: number;
  currency: string;
}

export interface OutboxRow {
  id: number;
  payload: PaymentEvent;
  status: OutboxStatus;
  created_at: string;
  published_at: string | null;
}

export interface BrokerState {
  down: boolean;
  received_count: number;
  received: PaymentEvent[];
}

export interface AppState {
  payments: Payment[];
  outbox: OutboxRow[];
  broker: BrokerState;
}

export interface CreatePaymentRequest {
  customer: string;
  amount: number;
  currency?: string;
}
