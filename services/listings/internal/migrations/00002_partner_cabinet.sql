-- +goose Up
-- Daily counters behind the partner cabinet: page views of a listing and
-- clicks on its contact buttons. Only counts are kept, never visitors.
CREATE TABLE listing_stats_daily (
    target_type     text NOT NULL CHECK (target_type IN ('hotel', 'local_guide', 'tour')),
    target_id       uuid NOT NULL,
    -- A Bishkek calendar day.
    day             date NOT NULL,
    views           integer NOT NULL DEFAULT 0,
    phone_clicks    integer NOT NULL DEFAULT 0,
    whatsapp_clicks integer NOT NULL DEFAULT 0,
    telegram_clicks integer NOT NULL DEFAULT 0,
    website_clicks  integer NOT NULL DEFAULT 0,
    PRIMARY KEY (target_type, target_id, day)
);

-- The listing owner's public answer to a review.
ALTER TABLE reviews ADD COLUMN reply text, ADD COLUMN replied_at timestamptz;

-- +goose Down
ALTER TABLE reviews DROP COLUMN reply, DROP COLUMN replied_at;
DROP TABLE listing_stats_daily;
