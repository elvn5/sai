-- +goose Up
-- The partner's CRM: their clients (those who signed up for their tours and
-- ones they add by hand), each participant's status and payments, and chats
-- between partners and travelers.

-- A partner's client: an account (user_id) that signed up for one of their
-- tours, or someone added by hand (user_id NULL: name and phone typed in).
CREATE TABLE crm_clients (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id uuid NOT NULL,
    user_id    uuid,
    name       text,
    phone      text,
    telegram   text,
    note       text,
    tags       text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (partner_id, user_id)
);
CREATE INDEX crm_clients_partner_idx ON crm_clients (partner_id, created_at DESC);

INSERT INTO crm_clients (partner_id, user_id, created_at)
SELECT t.user_id, p.user_id, min(p.joined_at)
FROM tour_participants p JOIN tours t ON t.id = p.tour_id
WHERE t.user_id IS NOT NULL
GROUP BY t.user_id, p.user_id;

-- A participant is an account that signed up (user_id) or a client the
-- partner added by hand (client_id). Only "new" and "confirmed" take a
-- place; "waitlist" waits for one, "cancelled" stays for the history.
ALTER TABLE tour_participants DROP CONSTRAINT tour_participants_pkey;
ALTER TABLE tour_participants
    ADD COLUMN id         uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    ALTER COLUMN user_id DROP NOT NULL,
    ADD COLUMN client_id  uuid REFERENCES crm_clients (id) ON DELETE CASCADE,
    ADD COLUMN status     text NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'confirmed', 'waitlist', 'cancelled')),
    -- What the participant owes; the tour's price when unset.
    ADD COLUMN amount_due integer CHECK (amount_due >= 0),
    ADD CONSTRAINT tour_participants_one_person CHECK ((user_id IS NULL) <> (client_id IS NULL));
UPDATE tour_participants SET status = 'confirmed';
CREATE UNIQUE INDEX tour_participants_user_key ON tour_participants (tour_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX tour_participants_client_key ON tour_participants (tour_id, client_id) WHERE client_id IS NOT NULL;
CREATE INDEX tour_participants_client_idx ON tour_participants (client_id);

-- Payments the partner received from a participant (cash, transfer...).
CREATE TABLE tour_payments (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id uuid NOT NULL REFERENCES tour_participants (id) ON DELETE CASCADE,
    amount         integer NOT NULL CHECK (amount > 0),
    method         text NOT NULL CHECK (method IN ('cash', 'transfer', 'card', 'other')),
    paid_on        date NOT NULL,
    note           text,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tour_payments_participant_idx ON tour_payments (participant_id);

-- One chat per partner and traveler.
CREATE TABLE chats (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id      uuid NOT NULL,
    user_id         uuid NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_message_at timestamptz,
    partner_read_at timestamptz,
    user_read_at    timestamptz,
    UNIQUE (partner_id, user_id)
);
CREATE INDEX chats_user_idx ON chats (user_id, last_message_at DESC);
CREATE INDEX chats_partner_idx ON chats (partner_id, last_message_at DESC);

CREATE TABLE chat_messages (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id    uuid NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    sender_id  uuid NOT NULL,
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX chat_messages_chat_idx ON chat_messages (chat_id, created_at DESC);

-- A notification may open a page of the site instead of pressing back to
-- the bot (a new chat message).
ALTER TABLE notifications ADD COLUMN button_url text;

-- +goose Down
ALTER TABLE notifications DROP COLUMN button_url;
DROP TABLE chat_messages;
DROP TABLE chats;
DROP TABLE tour_payments;
DELETE FROM tour_participants WHERE user_id IS NULL;
DROP INDEX tour_participants_user_key;
DROP INDEX tour_participants_client_key;
ALTER TABLE tour_participants DROP CONSTRAINT tour_participants_pkey;
ALTER TABLE tour_participants DROP COLUMN id, DROP COLUMN client_id, DROP COLUMN status, DROP COLUMN amount_due,
    ALTER COLUMN user_id SET NOT NULL, ADD PRIMARY KEY (tour_id, user_id);
DROP TABLE crm_clients;
