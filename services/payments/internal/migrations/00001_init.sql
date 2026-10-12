-- +goose Up
-- A booking someone pays for online. The item (a tour, later a hotel stay
-- or an outing fee) is snapshotted at booking time; user ids reference the
-- users service. Money is in whole som.
CREATE TABLE orders (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind           text NOT NULL CHECK (kind IN ('tour')),
    item_id        uuid NOT NULL,
    item_title     text NOT NULL,
    item_slug      text,
    item_locale    text,
    item_date      date,
    quantity       integer NOT NULL CHECK (quantity > 0),
    unit_price     integer NOT NULL CHECK (unit_price > 0),
    amount         integer NOT NULL CHECK (amount > 0),
    -- The platform's commission and what's owed to the partner.
    fee            integer NOT NULL DEFAULT 0,
    payee_amount   integer NOT NULL,
    user_id        uuid NOT NULL,
    payee_id       uuid NOT NULL,
    contact_name   text NOT NULL,
    contact_phone  text NOT NULL,
    comment        text,
    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'paid', 'failed', 'expired', 'refunded')),
    provider       text NOT NULL,
    payment_url    text,
    provider_error text,
    expires_at     timestamptz NOT NULL,
    checked_at     timestamptz,
    paid_at        timestamptz,
    -- Telegram messages about the payment went out.
    notified_at    timestamptz,
    refund_note    text,
    payout_id      uuid,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX orders_pending_idx ON orders (created_at) WHERE status = 'pending';
CREATE INDEX orders_user_idx ON orders (user_id, created_at DESC);
CREATE INDEX orders_payee_idx ON orders (payee_id, created_at DESC);

-- Money handed over to a partner for a set of paid orders.
CREATE TABLE payouts (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    payee_id   uuid NOT NULL,
    amount     integer NOT NULL,
    orders     integer NOT NULL,
    note       text,
    created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE orders ADD CONSTRAINT orders_payout_fk FOREIGN KEY (payout_id) REFERENCES payouts (id);

-- +goose Down
DROP TABLE orders;
DROP TABLE payouts;
