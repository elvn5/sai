-- +goose Up
-- The organizer marks who turned up at the meeting point (in the bot).
ALTER TABLE outing_participants ADD COLUMN arrived_at timestamptz;

-- +goose Down
ALTER TABLE outing_participants DROP COLUMN arrived_at;
