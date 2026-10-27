-- +goose Up
-- Sessions (user_token JWTs) issued before this are refused: set when the
-- password is reset, so a reset signs every other device out.
ALTER TABLE users ADD COLUMN sessions_valid_after timestamptz;

-- One-time links for setting a new password, sent by email or to the
-- linked Telegram chat. Only the sha256 of the token is kept.
CREATE TABLE password_reset_tokens (
    token_hash text PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    channel    text NOT NULL CHECK (channel IN ('email', 'telegram')),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);

-- Every reset request, whether or not the account exists, for rate limits
-- by what was typed (its sha256) and by IP. Rows older than a day go.
CREATE TABLE password_reset_requests (
    id              bigserial PRIMARY KEY,
    identifier_hash text NOT NULL,
    ip              text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX password_reset_requests_identifier_idx ON password_reset_requests (identifier_hash, created_at);
CREATE INDEX password_reset_requests_ip_idx ON password_reset_requests (ip, created_at);

-- +goose Down
DROP TABLE password_reset_requests;
DROP TABLE password_reset_tokens;
ALTER TABLE users DROP COLUMN sessions_valid_after;
