-- +goose Up
-- Chats that get new blog posts.
CREATE TABLE blog_subscribers (
    chat_id    bigint PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Posts the feed has already seen (sent, or there before it started).
CREATE TABLE blog_seen (
    post_id text PRIMARY KEY,
    seen_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE blog_seen;
DROP TABLE blog_subscribers;
