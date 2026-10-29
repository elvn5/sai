-- +goose Up
-- More of the partner's CRM (docs/crm-roadmap.md): refunds, a tour's
-- expenses and staff, where clients come from, leads, tasks, message
-- templates and broadcasts.

-- A refund to the participant subtracts from what they paid.
ALTER TABLE tour_payments ADD COLUMN kind text NOT NULL DEFAULT 'payment' CHECK (kind IN ('payment', 'refund'));

-- Where a client came from.
ALTER TABLE crm_clients ADD COLUMN source text
    CHECK (source IN ('ermekten', 'instagram', 'whatsapp', 'telegram', 'phone', 'referral', 'other'));
UPDATE crm_clients SET source = 'ermekten' WHERE user_id IS NOT NULL;

-- What a departure cost the partner.
CREATE TABLE tour_expenses (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tour_id    uuid NOT NULL REFERENCES tours (id) ON DELETE CASCADE,
    category   text NOT NULL CHECK (category IN ('transport', 'guides', 'food', 'accommodation', 'permits', 'gear', 'other')),
    amount     integer NOT NULL CHECK (amount > 0),
    note       text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tour_expenses_tour_idx ON tour_expenses (tour_id);

-- Who leads the group and drives it.
CREATE TABLE tour_staff (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tour_id    uuid NOT NULL REFERENCES tours (id) ON DELETE CASCADE,
    role       text NOT NULL CHECK (role IN ('guide', 'assistant', 'driver', 'cook', 'other')),
    name       text NOT NULL,
    phone      text,
    note       text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tour_staff_tour_idx ON tour_staff (tour_id);

-- An inquiry on its way to a booking. Opened by hand or when a traveler
-- first writes to the partner; a sign-up for the partner's tour wins it.
CREATE TABLE crm_leads (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id  uuid NOT NULL,
    client_id   uuid NOT NULL REFERENCES crm_clients (id) ON DELETE CASCADE,
    tour_id     uuid REFERENCES tours (id) ON DELETE SET NULL,
    stage       text NOT NULL DEFAULT 'new' CHECK (stage IN ('new', 'working', 'offer', 'won', 'lost')),
    lost_reason text,
    note        text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX crm_leads_partner_idx ON crm_leads (partner_id, updated_at DESC);
CREATE INDEX crm_leads_client_idx ON crm_leads (client_id);

-- "Call back tomorrow", "remind about the balance".
CREATE TABLE crm_tasks (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id uuid NOT NULL,
    client_id  uuid REFERENCES crm_clients (id) ON DELETE CASCADE,
    tour_id    uuid REFERENCES tours (id) ON DELETE CASCADE,
    title      text NOT NULL,
    due_on     date NOT NULL,
    done_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX crm_tasks_partner_idx ON crm_tasks (partner_id, due_on) WHERE done_at IS NULL;

-- Texts the partner sends again and again.
CREATE TABLE crm_templates (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id uuid NOT NULL,
    title      text NOT NULL,
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX crm_templates_partner_idx ON crm_templates (partner_id, title);

-- A message sent to many clients at once (each gets it in their chat).
CREATE TABLE crm_broadcasts (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id uuid NOT NULL,
    audience   text NOT NULL,
    body       text NOT NULL,
    sent       integer NOT NULL,
    skipped    integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX crm_broadcasts_partner_idx ON crm_broadcasts (partner_id, created_at DESC);

-- +goose Down
DROP TABLE crm_broadcasts;
DROP TABLE crm_templates;
DROP TABLE crm_tasks;
DROP TABLE crm_leads;
DROP TABLE tour_staff;
DROP TABLE tour_expenses;
ALTER TABLE crm_clients DROP COLUMN source;
ALTER TABLE tour_payments DROP COLUMN kind;
