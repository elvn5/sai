-- +goose Up
-- The admin panel's history: each change an admin session made here
-- (services/pkg/audit). Times are kept to the millisecond, as the panel
-- sees them, so its "before" cursor pages without gaps.
CREATE TABLE admin_history (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    module     text NOT NULL,
    method     text NOT NULL,
    route      text NOT NULL,
    path       text NOT NULL,
    entity_id  text,
    subject    text,
    payload    jsonb,
    status     integer NOT NULL,
    ip         text NOT NULL DEFAULT '',
    user_agent text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT date_trunc('milliseconds', now())
);
CREATE INDEX admin_history_module_created_idx ON admin_history (module, created_at DESC);

-- +goose Down
DROP TABLE admin_history;
