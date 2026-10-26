-- +goose Up
-- Travelers sign up for a tour in the site or the bot (what the trails
-- service's outings did): the partner sees who signed up, marks who came,
-- and participants rate the tour after it (a review). The meeting point and
-- time (HH:MM, Bishkek, on start_date) say where and when it starts.
ALTER TABLE tours
    ADD COLUMN meeting_point    text,
    ADD COLUMN meeting_time     text,
    -- When the reminder and the "how did it go?" message are due (computed
    -- by models.Tour.Schedule) and when they went out.
    ADD COLUMN remind_at        timestamptz,
    ADD COLUMN followup_at      timestamptz,
    ADD COLUMN reminder_sent_at timestamptz,
    ADD COLUMN followup_sent_at timestamptz;
CREATE INDEX tours_remind_at_idx ON tours (remind_at) WHERE reminder_sent_at IS NULL;
CREATE INDEX tours_followup_at_idx ON tours (followup_at) WHERE followup_sent_at IS NULL;

CREATE TABLE tour_participants (
    tour_id    uuid NOT NULL REFERENCES tours (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL,
    joined_at  timestamptz NOT NULL DEFAULT now(),
    -- Set by the partner at the meeting point.
    arrived_at timestamptz,
    PRIMARY KEY (tour_id, user_id)
);
CREATE INDEX tour_participants_user_idx ON tour_participants (user_id);

-- Outbox of Telegram notifications about sign-ups, delivered by a
-- background worker (internal/notify) so requests never wait on Telegram.
-- tour_id has no foreign key: a deleted tour's notice still goes out.
CREATE TABLE notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL,
    kind        text NOT NULL,
    tour_id     uuid,
    text        text NOT NULL,
    button_text text,
    -- Callback data the bot answers.
    button_data text,
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

-- +goose Down
DROP TABLE notifications;
DROP TABLE tour_participants;
ALTER TABLE tours DROP COLUMN meeting_point, DROP COLUMN meeting_time, DROP COLUMN remind_at,
    DROP COLUMN followup_at, DROP COLUMN reminder_sent_at, DROP COLUMN followup_sent_at;
