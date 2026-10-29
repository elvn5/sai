-- +goose Up
-- A participant's form, filled at sign-up (or later, or by the partner for
-- a client added by hand): an emergency contact, mountain experience and
-- health notes the organizer should know; and when they accepted the
-- platform's rules and risks of the trip (waiver_version: which text).
ALTER TABLE tour_participants
    ADD COLUMN form             jsonb,
    ADD COLUMN form_at          timestamptz,
    ADD COLUMN waiver_version   text,
    ADD COLUMN waiver_at        timestamptz;

-- +goose Down
ALTER TABLE tour_participants DROP COLUMN form, DROP COLUMN form_at, DROP COLUMN waiver_version, DROP COLUMN waiver_at;
