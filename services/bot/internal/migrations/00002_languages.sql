-- +goose Up
-- The language each chat reads the bot in: picked by the user, or taken
-- from their Telegram app the first time.
CREATE TABLE chat_languages (
    chat_id    bigint PRIMARY KEY,
    lang       text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE chat_languages;
