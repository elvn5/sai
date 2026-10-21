-- +goose Up
-- Participants rate an outing (and so its organizer) after it: one rating
-- each, changeable. organizer_id is copied from the outing so the
-- organizer's rating is one indexed read.
CREATE TABLE outing_ratings (
    outing_id    uuid NOT NULL REFERENCES outings (id) ON DELETE CASCADE,
    user_id      uuid NOT NULL,
    organizer_id uuid NOT NULL,
    rating       integer NOT NULL CHECK (rating BETWEEN 1 AND 5),
    text         text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (outing_id, user_id)
);
CREATE INDEX outing_ratings_organizer_idx ON outing_ratings (organizer_id, created_at DESC);

-- +goose Down
DROP TABLE outing_ratings;
