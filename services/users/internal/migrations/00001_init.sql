-- +goose Up
CREATE TABLE users (
    id                            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_id                   bigint UNIQUE,
    email                         text UNIQUE,
    password_hash                 text,
    username                      text,
    first_name                    text,
    last_name                     text,
    phone                         text UNIQUE,
    avatar_url                    text,
    -- Email/phone registrations start unverified; Telegram accounts don't.
    is_verified                   boolean NOT NULL DEFAULT true,
    -- sha256 of the pending email code or Telegram deep-link token.
    verification_code_hash        text,
    verification_code_expires_at  timestamptz,
    verification_code_sent_at     timestamptz,
    verification_attempts         integer NOT NULL DEFAULT 0,
    partner_status                text NOT NULL DEFAULT 'none'
                                  CHECK (partner_status IN ('none', 'pending', 'approved', 'rejected')),
    partner_requested_at          timestamptz,
    created_at                    timestamptz NOT NULL DEFAULT now(),
    updated_at                    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX users_verification_code_hash_idx ON users (verification_code_hash)
    WHERE verification_code_hash IS NOT NULL;

-- Single row: the support contacts the admin panel edits and the bot shows.
CREATE TABLE support_info (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    message          text NOT NULL,
    phone            text,
    telegram_contact text,
    email            text,
    working_hours    text,
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE legacy_imports (
    name        text PRIMARY KEY,
    imported_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE legacy_imports;
DROP TABLE support_info;
DROP TABLE users;
