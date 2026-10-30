-- +goose Up
-- The gateway's record of each order's payment (GoPay): its id, its own
-- status and times, as last reported by the gateway — the admin panel's
-- online payments. One payment per order.
CREATE TABLE provider_payments (
    order_id           uuid PRIMARY KEY REFERENCES orders (id),
    provider           text NOT NULL,
    payment_id         text,
    -- The order id as the gateway knows it (GoPay: the uuid without dashes).
    provider_order_id  text NOT NULL,
    status             text NOT NULL,
    amount             numeric(12, 2) NOT NULL,
    testing            boolean NOT NULL DEFAULT false,
    checkout_url       text,
    gateway_created_at timestamptz,
    expires_at         timestamptz,
    committed_at       timestamptz,
    bank_op_date       date,
    -- The last failed request to the gateway about it.
    last_error         text,
    checked_at         timestamptz NOT NULL DEFAULT now(),
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX provider_payments_payment_idx ON provider_payments (provider, payment_id);
CREATE INDEX provider_payments_created_idx ON provider_payments (created_at DESC);

-- Webhook deliveries from the gateway. event_key (event + payment) makes a
-- repeated delivery of a processed event a no-op.
CREATE TABLE provider_events (
    id           bigserial PRIMARY KEY,
    provider     text NOT NULL,
    event_key    text NOT NULL,
    event        text NOT NULL,
    payment_id   text,
    order_id     uuid,
    testing      boolean NOT NULL DEFAULT false,
    payload      jsonb NOT NULL,
    attempts     integer NOT NULL DEFAULT 1,
    received_at  timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    error        text,
    UNIQUE (provider, event_key)
);
CREATE INDEX provider_events_order_idx ON provider_events (order_id, received_at DESC);

-- +goose Down
DROP TABLE provider_events;
DROP TABLE provider_payments;
