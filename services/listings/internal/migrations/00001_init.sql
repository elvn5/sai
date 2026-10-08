-- +goose Up
CREATE TABLE hotels (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                text NOT NULL UNIQUE,
    name                text NOT NULL,
    type                text NOT NULL DEFAULT 'hotel',
    description         text NOT NULL,
    destination         text NOT NULL,
    address             text,
    latitude            double precision,
    longitude           double precision,
    price_from          integer,
    stars               smallint,
    amenities           text[] NOT NULL DEFAULT '{}',
    cancellation_policy text NOT NULL DEFAULT 'flexible',
    check_in_time       text,
    check_out_time      text,
    cover_image_url     text,
    photos              text[] NOT NULL DEFAULT '{}',
    phone               text,
    whatsapp            text,
    telegram            text,
    website             text,
    -- Denormalized from reviews for filtering and sorting.
    rating_average      real NOT NULL DEFAULT 0,
    review_count        integer NOT NULL DEFAULT 0,
    is_published        boolean NOT NULL DEFAULT true,
    -- Owner in the users service (no cross-database foreign key).
    user_id             uuid,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX hotels_destination_idx ON hotels (destination);
CREATE INDEX hotels_user_id_idx ON hotels (user_id);

-- People offering guide services (not the written guides of the content
-- service). One profile per user.
CREATE TABLE local_guides (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug             text NOT NULL UNIQUE,
    name             text NOT NULL,
    headline         text,
    bio              text NOT NULL,
    avatar_url       text,
    cover_image_url  text,
    photos           text[] NOT NULL DEFAULT '{}',
    languages        text[] NOT NULL DEFAULT '{}',
    specializations  text[] NOT NULL DEFAULT '{}',
    destinations     text[] NOT NULL DEFAULT '{}',
    price_per_day    integer,
    experience_years smallint,
    has_vehicle      boolean NOT NULL DEFAULT false,
    phone            text,
    whatsapp         text,
    telegram         text,
    rating_average   real NOT NULL DEFAULT 0,
    review_count     integer NOT NULL DEFAULT 0,
    is_published     boolean NOT NULL DEFAULT true,
    user_id          uuid UNIQUE,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tours (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    locale              text NOT NULL CHECK (locale IN ('ru', 'ky', 'en', 'de')),
    slug                text NOT NULL,
    title               text NOT NULL,
    excerpt             text,
    content             text NOT NULL,
    price               integer,
    duration_days       integer,
    location            text,
    start_date          date NOT NULL,
    end_date            date NOT NULL,
    destination         text,
    cancellation_policy text,
    cover_image_url     text,
    phone               text,
    whatsapp            text,
    telegram            text,
    rating_average      real NOT NULL DEFAULT 0,
    review_count        integer NOT NULL DEFAULT 0,
    is_published        boolean NOT NULL DEFAULT false,
    published_at        timestamptz,
    user_id             uuid,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (locale, slug)
);
CREATE INDEX tours_user_id_idx ON tours (user_id);

-- Polymorphic: target_id is a hotel, local guide or tour per target_type.
CREATE TABLE reviews (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL CHECK (target_type IN ('hotel', 'local_guide', 'tour')),
    target_id   uuid NOT NULL,
    rating      smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
    text        text NOT NULL,
    photos      text[] NOT NULL DEFAULT '{}',
    user_id     uuid NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (target_type, target_id, user_id)
);
CREATE INDEX reviews_target_idx ON reviews (target_type, target_id, created_at DESC);

-- News/photo updates an owner posts on their listing.
CREATE TABLE listing_posts (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL CHECK (target_type IN ('hotel', 'local_guide', 'tour')),
    target_id   uuid NOT NULL,
    text        text NOT NULL,
    photos      text[] NOT NULL DEFAULT '{}',
    user_id     uuid NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX listing_posts_target_idx ON listing_posts (target_type, target_id, created_at DESC);

CREATE TABLE legacy_imports (
    name        text PRIMARY KEY,
    imported_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE legacy_imports;
DROP TABLE listing_posts;
DROP TABLE reviews;
DROP TABLE tours;
DROP TABLE local_guides;
DROP TABLE hotels;
