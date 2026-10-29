-- +goose Up
-- The driver's CRM (a partner whose kind in users is "driver"): their card
-- in the organizers' catalog, vehicles, trips, what customers paid and
-- what the vehicles cost. Customers are the driver's crm_clients: tour
-- organizers who asked for transport (with an account) and people added
-- by hand.

-- What organizers see when they look for transport.
CREATE TABLE driver_profiles (
    user_id    uuid PRIMARY KEY,
    name       text NOT NULL,
    phone      text,
    region     text,
    about      text,
    -- Shown in the organizers' catalog.
    published  boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE driver_vehicles (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id        uuid NOT NULL,
    kind             text NOT NULL CHECK (kind IN ('sedan', 'minivan', 'minibus', 'bus', 'offroad', 'other')),
    model            text NOT NULL,
    plate            text,
    seats            integer NOT NULL CHECK (seats BETWEEN 1 AND 80),
    year             integer CHECK (year BETWEEN 1950 AND 2100),
    -- When the insurance and the roadworthiness check run out.
    insurance_until  date,
    inspection_until date,
    note             text,
    active           boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX driver_vehicles_driver_idx ON driver_vehicles (driver_id);

-- A trip: asked for by an organizer for one of their tours (tour_id set,
-- status "requested" until the driver answers) or written down by the
-- driver themself.
CREATE TABLE driver_trips (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id      uuid NOT NULL,
    vehicle_id     uuid REFERENCES driver_vehicles (id) ON DELETE SET NULL,
    client_id      uuid REFERENCES crm_clients (id) ON DELETE SET NULL,
    tour_id        uuid REFERENCES tours (id) ON DELETE SET NULL,
    status         text NOT NULL DEFAULT 'accepted'
                   CHECK (status IN ('requested', 'accepted', 'declined', 'done', 'cancelled')),
    starts_at      timestamptz NOT NULL,
    ends_at        timestamptz,
    route_from     text NOT NULL,
    route_to       text NOT NULL,
    passengers     integer CHECK (passengers > 0),
    price          integer CHECK (price >= 0),
    note           text,
    decline_reason text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at IS NULL OR ends_at >= starts_at)
);
CREATE INDEX driver_trips_driver_idx ON driver_trips (driver_id, starts_at);
CREATE INDEX driver_trips_tour_idx ON driver_trips (tour_id) WHERE tour_id IS NOT NULL;

CREATE TABLE driver_trip_payments (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id    uuid NOT NULL REFERENCES driver_trips (id) ON DELETE CASCADE,
    amount     integer NOT NULL CHECK (amount > 0),
    method     text NOT NULL CHECK (method IN ('cash', 'transfer', 'card', 'other')),
    paid_on    date NOT NULL,
    note       text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX driver_trip_payments_trip_idx ON driver_trip_payments (trip_id);

-- Fuel, repairs and the rest, of a vehicle or a trip.
CREATE TABLE driver_expenses (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id  uuid NOT NULL,
    vehicle_id uuid REFERENCES driver_vehicles (id) ON DELETE SET NULL,
    trip_id    uuid REFERENCES driver_trips (id) ON DELETE SET NULL,
    category   text NOT NULL CHECK (category IN ('fuel', 'repair', 'maintenance', 'wash', 'parking', 'fines', 'insurance', 'other')),
    amount     integer NOT NULL CHECK (amount > 0),
    spent_on   date NOT NULL,
    note       text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX driver_expenses_driver_idx ON driver_expenses (driver_id, spent_on DESC);

-- +goose Down
DROP TABLE driver_expenses;
DROP TABLE driver_trip_payments;
DROP TABLE driver_trips;
DROP TABLE driver_vehicles;
DROP TABLE driver_profiles;
