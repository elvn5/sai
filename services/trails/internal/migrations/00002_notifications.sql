-- +goose Up
-- Outbox of Telegram notifications, delivered by a background worker
-- (internal/notify) so requests never wait on Telegram.
CREATE TABLE notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL,
    kind        text NOT NULL,
    outing_id   uuid REFERENCES outings (id) ON DELETE CASCADE,
    text        text NOT NULL,
    button_text text,
    button_url  text,
    -- Set for one-off messages (reminders) so they're never queued twice.
    dedup_key   text UNIQUE,
    status      text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'skipped', 'failed')),
    attempts    integer NOT NULL DEFAULT 0,
    last_error  text,
    send_after  timestamptz NOT NULL DEFAULT now(),
    sent_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notifications_pending_idx ON notifications (send_after) WHERE status = 'pending';

-- When the reminder and the "how did it go?" message are due (computed by
-- models.Outing.Schedule; null: not sent for this outing), and when they
-- went out.
ALTER TABLE outings ADD COLUMN remind_at timestamptz;
ALTER TABLE outings ADD COLUMN followup_at timestamptz;
ALTER TABLE outings ADD COLUMN reminder_sent_at timestamptz;
ALTER TABLE outings ADD COLUMN followup_sent_at timestamptz;
CREATE INDEX outings_remind_at_idx ON outings (remind_at) WHERE reminder_sent_at IS NULL;
CREATE INDEX outings_followup_at_idx ON outings (followup_at) WHERE followup_sent_at IS NULL;

-- +goose Down
ALTER TABLE outings DROP COLUMN followup_sent_at;
ALTER TABLE outings DROP COLUMN reminder_sent_at;
ALTER TABLE outings DROP COLUMN followup_at;
ALTER TABLE outings DROP COLUMN remind_at;
DROP TABLE notifications;
