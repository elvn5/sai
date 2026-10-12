-- +goose Up
-- A pending request to link a Telegram account to this (site) account: the
-- sha256 of the token in the t.me/<bot>?start=link_<token> deep link.
ALTER TABLE users ADD COLUMN telegram_link_token_hash text;
ALTER TABLE users ADD COLUMN telegram_link_expires_at timestamptz;
CREATE UNIQUE INDEX users_telegram_link_token_hash_idx ON users (telegram_link_token_hash)
    WHERE telegram_link_token_hash IS NOT NULL;

-- +goose Down
DROP INDEX users_telegram_link_token_hash_idx;
ALTER TABLE users DROP COLUMN telegram_link_expires_at;
ALTER TABLE users DROP COLUMN telegram_link_token_hash;
