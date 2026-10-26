-- +goose Up
-- Outings merged into tours (the listings service signs travelers up for
-- tours now); the old outings are dropped.
DROP TABLE notifications;
DROP TABLE outing_ratings;
DROP TABLE outing_participants;
DROP TABLE outings;

-- +goose Down
CREATE TABLE outings (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trail_id        uuid REFERENCES trails (id) ON DELETE SET NULL,
    title           text NOT NULL,
    starts_at       timestamptz NOT NULL,
    returns_at      timestamptz,
    meeting_point   text NOT NULL,
    transport       text NOT NULL,
    seats           integer NOT NULL,
    cost_per_person integer,
    level           text NOT NULL,
    description     text,
    contact         text,
    status          text NOT NULL DEFAULT 'open',
    user_id         uuid NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    remind_at        timestamptz,
    followup_at      timestamptz,
    reminder_sent_at timestamptz,
    followup_sent_at timestamptz
);
CREATE TABLE outing_participants (
    outing_id  uuid NOT NULL REFERENCES outings (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL,
    joined_at  timestamptz NOT NULL DEFAULT now(),
    arrived_at timestamptz,
    PRIMARY KEY (outing_id, user_id)
);
CREATE TABLE outing_ratings (
    outing_id    uuid NOT NULL REFERENCES outings (id) ON DELETE CASCADE,
    user_id      uuid NOT NULL,
    organizer_id uuid NOT NULL,
    rating       integer NOT NULL CHECK (rating BETWEEN 1 AND 5),
    text         text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (outing_id, user_id)
);
CREATE TABLE notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL,
    kind        text NOT NULL,
    outing_id   uuid REFERENCES outings (id) ON DELETE CASCADE,
    text        text NOT NULL,
    button_text text,
    button_url  text,
    button_data text,
    dedup_key   text UNIQUE,
    status      text NOT NULL DEFAULT 'pending',
    attempts    integer NOT NULL DEFAULT 0,
    last_error  text,
    send_after  timestamptz NOT NULL DEFAULT now(),
    sent_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
