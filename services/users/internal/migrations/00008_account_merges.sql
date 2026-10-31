-- +goose Up
-- Accounts merged into another account of the same person (a Telegram-only
-- one into the account with their email or phone). The merged account is
-- deleted; other services poll this list (ListAccountMerges) and move what
-- it owned in their databases.
CREATE TABLE account_merges (
    id         bigserial PRIMARY KEY,
    from_id    uuid NOT NULL,
    to_id      uuid NOT NULL,
    reason     text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- An email being added to a signed-in account: it becomes the account's
-- email once the code sent to it (verification_code_*) is typed.
ALTER TABLE users ADD COLUMN pending_email text;

-- +goose Down
ALTER TABLE users DROP COLUMN pending_email;
DROP TABLE account_merges;
