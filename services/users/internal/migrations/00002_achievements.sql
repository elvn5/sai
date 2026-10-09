-- +goose Up
-- The code a user shows a partner (as text or a QR link) to receive an
-- achievement. Generated on first use.
ALTER TABLE users ADD COLUMN award_code text UNIQUE;

-- The catalog the admin panel manages. Texts are keyed by locale
-- (ru is required; the site falls back to it).
CREATE TABLE achievements (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title       jsonb NOT NULL,
    description jsonb NOT NULL DEFAULT '{}',
    icon        text NOT NULL,
    rarity      text NOT NULL DEFAULT 'common'
                CHECK (rarity IN ('common', 'rare', 'epic', 'legendary')),
    is_active   boolean NOT NULL DEFAULT true,
    sort_order  integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- An achievement a partner handed to a user; each user earns each
-- achievement once.
CREATE TABLE user_achievements (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    achievement_id uuid NOT NULL REFERENCES achievements (id) ON DELETE RESTRICT,
    awarded_by     uuid REFERENCES users (id) ON DELETE SET NULL,
    note           text,
    awarded_at     timestamptz NOT NULL DEFAULT now(),
    -- When the user first saw it in the hall of fame (null: still new).
    seen_at        timestamptz,
    UNIQUE (user_id, achievement_id)
);
CREATE INDEX user_achievements_awarded_by_idx ON user_achievements (awarded_by, awarded_at DESC);
CREATE INDEX user_achievements_awarded_at_idx ON user_achievements (awarded_at DESC);

-- +goose Down
DROP TABLE user_achievements;
DROP TABLE achievements;
ALTER TABLE users DROP COLUMN award_code;
