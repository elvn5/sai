-- +goose Up
-- What kind of partner an account is: a tour organizer or a driver who
-- carries tour groups (the driver's CRM lives in listings).
ALTER TABLE users ADD COLUMN partner_kind text NOT NULL DEFAULT 'organizer'
    CHECK (partner_kind IN ('organizer', 'driver'));

-- +goose Down
ALTER TABLE users DROP COLUMN partner_kind;
