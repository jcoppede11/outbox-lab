CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE payments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer   TEXT          NOT NULL,
    amount     NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    currency   CHAR(3)       NOT NULL DEFAULT 'USD',
    status     TEXT          NOT NULL DEFAULT 'created'
                   CHECK (status IN ('created', 'authorized', 'failed')),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE TABLE outbox (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    payload      JSONB       NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_outbox_pending ON outbox (id) WHERE status = 'pending';
