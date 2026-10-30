-- +goose Up
-- A traveler's participant form kept in their profile (emergency contact,
-- mountain experience, health): organizers of the tours they sign up for
-- see it wherever the tour has no form of its own for them.
CREATE TABLE participant_profiles (
    user_id    uuid PRIMARY KEY,
    form       jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE participant_profiles;
