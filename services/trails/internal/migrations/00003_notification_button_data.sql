-- +goose Up
-- A notification's button can press back to the bot (callback data its
-- handlers answer: the outing's card, a condition report) instead of
-- opening a URL.
ALTER TABLE notifications ADD COLUMN button_data text;

-- +goose Down
ALTER TABLE notifications DROP COLUMN button_data;
