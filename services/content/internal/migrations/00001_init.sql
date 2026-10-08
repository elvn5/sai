-- +goose Up
CREATE TABLE blog_posts (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    locale          text NOT NULL CHECK (locale IN ('ru', 'ky', 'en', 'de')),
    slug            text NOT NULL,
    title           text NOT NULL,
    excerpt         text,
    content         text NOT NULL,
    cover_image_url text,
    seo_title       text,
    seo_description text,
    is_published    boolean NOT NULL DEFAULT false,
    published_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (locale, slug)
);

CREATE TABLE guides (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    locale              text NOT NULL CHECK (locale IN ('ru', 'ky', 'en', 'de')),
    slug                text NOT NULL,
    title               text NOT NULL,
    excerpt             text,
    content             text NOT NULL,
    difficulty          text NOT NULL DEFAULT 'easy' CHECK (difficulty IN ('easy', 'moderate', 'hard')),
    season              text NOT NULL DEFAULT 'all_year' CHECK (season IN ('summer', 'winter', 'all_year')),
    equipment_checklist text[] NOT NULL DEFAULT '{}',
    cover_image_url     text,
    attachments         jsonb NOT NULL DEFAULT '[]',
    seo_title           text,
    seo_description     text,
    is_published        boolean NOT NULL DEFAULT false,
    published_at        timestamptz,
    -- Owner in the users service (no cross-database foreign key).
    user_id             uuid,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (locale, slug)
);
CREATE INDEX guides_user_id_idx ON guides (user_id);

CREATE TABLE gallery_items (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    image_url    text NOT NULL,
    location     text,
    caption      text,
    alt_text     text,
    sort_order   integer NOT NULL DEFAULT 0,
    is_published boolean NOT NULL DEFAULT true,
    user_id      uuid,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX gallery_items_feed_idx ON gallery_items (is_published, sort_order, created_at DESC);
CREATE INDEX gallery_items_user_id_idx ON gallery_items (user_id);

CREATE TABLE page_seo (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    page         text NOT NULL CHECK (page IN ('home', 'blog', 'gallery', 'guides', 'tours')),
    locale       text NOT NULL CHECK (locale IN ('ru', 'ky', 'en', 'de')),
    title        text,
    description  text,
    og_image_url text,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (page, locale)
);

-- Records one-time data imports (see internal/legacyimport).
CREATE TABLE legacy_imports (
    name        text PRIMARY KEY,
    imported_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE legacy_imports;
DROP TABLE page_seo;
DROP TABLE gallery_items;
DROP TABLE guides;
DROP TABLE blog_posts;
