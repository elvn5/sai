-- +goose Up
-- A tour's photo gallery, its day-by-day program ([{"title": "Бишкек → Тамга",
-- "text": "…"}, …]) and how many places it has (null: not limited).
ALTER TABLE tours
    ADD COLUMN photos   text[] NOT NULL DEFAULT '{}',
    ADD COLUMN program  jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN capacity integer CHECK (capacity > 0);

-- Listings a traveler saved. User ids reference the users service.
CREATE TABLE favorites (
    user_id     uuid NOT NULL,
    target_type text NOT NULL CHECK (target_type IN ('hotel', 'local_guide', 'tour')),
    target_id   uuid NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, target_type, target_id)
);
CREATE INDEX favorites_target_idx ON favorites (target_type, target_id);

-- "Tell me about new tours": a Telegram message when a tour is published,
-- for one destination or ('') any.
CREATE TABLE tour_alerts (
    user_id     uuid NOT NULL,
    destination text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, destination)
);

-- +goose Down
DROP TABLE tour_alerts;
DROP TABLE favorites;
ALTER TABLE tours DROP COLUMN photos, DROP COLUMN program, DROP COLUMN capacity;
