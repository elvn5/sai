-- +goose Up
-- Partners publish blog posts too; admin-written posts have no user.
ALTER TABLE blog_posts ADD COLUMN user_id uuid;
CREATE INDEX blog_posts_user_id_idx ON blog_posts (user_id);

-- +goose Down
DROP INDEX blog_posts_user_id_idx;
ALTER TABLE blog_posts DROP COLUMN user_id;
