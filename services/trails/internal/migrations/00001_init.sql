-- +goose Up
-- The route catalog: hikes, treks and climbs, published by partners (or
-- an admin). User ids reference the users service (no foreign key).
CREATE TABLE trails (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            text NOT NULL UNIQUE,
    name            text NOT NULL,
    summary         text,
    description     text NOT NULL DEFAULT '',
    area            text NOT NULL,
    kind            text NOT NULL,
    grade           text NOT NULL,
    seasons         text[] NOT NULL DEFAULT '{}',
    gear            text[] NOT NULL DEFAULT '{}',
    summit_elevation integer,
    start_elevation  integer,
    elevation_gain   integer,
    distance_km      numeric(6, 1),
    duration_hours   numeric(5, 1),
    days             integer NOT NULL DEFAULT 1,
    -- [{"from": "КПП", "to": "Альплагерь", "hours": 1.5}, ...]
    segments        jsonb NOT NULL DEFAULT '[]',
    getting_there   text,
    start_latitude  double precision,
    start_longitude double precision,
    -- [[lat, lng, elevation|null], ...], downsampled from the GPX file.
    track           jsonb NOT NULL DEFAULT '[]',
    gpx_url         text,
    cover_image_url text,
    photos          text[] NOT NULL DEFAULT '{}',
    is_published    boolean NOT NULL DEFAULT true,
    user_id         uuid,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX trails_area_idx ON trails (area) WHERE is_published;
CREATE INDEX trails_user_id_idx ON trails (user_id);

-- What someone saw on a route on a given day.
CREATE TABLE condition_reports (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trail_id      uuid NOT NULL REFERENCES trails (id) ON DELETE CASCADE,
    user_id       uuid NOT NULL,
    observed_on   date NOT NULL,
    trail_state   text NOT NULL,
    traction      text NOT NULL,
    avalanche_risk text,
    snow_line     integer,
    reached       text,
    text          text,
    photos        text[] NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX condition_reports_trail_idx ON condition_reports (trail_id, observed_on DESC, created_at DESC);
CREATE INDEX condition_reports_recent_idx ON condition_reports (observed_on DESC, created_at DESC);

-- Group outings anyone signed in can organize and join.
CREATE TABLE outings (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trail_id        uuid REFERENCES trails (id) ON DELETE SET NULL,
    title           text NOT NULL,
    starts_at       timestamptz NOT NULL,
    returns_at      timestamptz,
    meeting_point   text NOT NULL,
    transport       text NOT NULL,
    seats           integer NOT NULL,
    cost_per_person integer,
    level           text NOT NULL,
    description     text,
    -- Shown to participants only.
    contact         text,
    status          text NOT NULL DEFAULT 'open',
    user_id         uuid NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX outings_starts_at_idx ON outings (starts_at);
CREATE INDEX outings_user_id_idx ON outings (user_id);

CREATE TABLE outing_participants (
    outing_id uuid NOT NULL REFERENCES outings (id) ON DELETE CASCADE,
    user_id   uuid NOT NULL,
    joined_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (outing_id, user_id)
);
CREATE INDEX outing_participants_user_idx ON outing_participants (user_id);

-- +goose Down
DROP TABLE outing_participants;
DROP TABLE outings;
DROP TABLE condition_reports;
DROP TABLE trails;
