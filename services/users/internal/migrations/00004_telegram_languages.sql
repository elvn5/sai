-- +goose Up
-- The language each Telegram chat reads the bot in, as the bot reports it,
-- for the notifications other services send. Keyed by Telegram id, not by
-- account: the bot may report it before the account exists, and it stays
-- with the Telegram id when the id moves to another account.
CREATE TABLE telegram_languages (
    telegram_id bigint PRIMARY KEY,
    lang        text NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE telegram_languages;
