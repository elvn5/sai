package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/content/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

const blogColumns = `id, locale, slug, title, excerpt, content, cover_image_url, seo_title,
	seo_description, difficulty, duration, season, is_published, published_at, user_id::text, created_at, updated_at`

func scanBlogPost(row pgx.Row) (models.BlogPost, error) {
	var p models.BlogPost
	var publishedAt *time.Time
	var createdAt, updatedAt time.Time
	err := row.Scan(&p.ID, &p.Locale, &p.Slug, &p.Title, &p.Excerpt, &p.Content, &p.CoverImageURL,
		&p.SeoTitle, &p.SeoDescription, &p.Difficulty, &p.Duration, &p.Season, &p.IsPublished, &publishedAt, &p.UserID, &createdAt, &updatedAt)
	p.PublishedAt, p.CreatedAt, p.UpdatedAt = timePtr(publishedAt), jsontime.New(createdAt), jsontime.New(updatedAt)
	return p, err
}

func (s *Store) collectBlogPosts(ctx context.Context, query string, args ...any) ([]models.BlogPost, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	posts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.BlogPost, error) { return scanBlogPost(row) })
	if err != nil {
		return nil, err
	}
	return posts, nil
}

// PublishedBlogPosts lists published posts, newest first. A nil locale
// means every locale.
func (s *Store) PublishedBlogPosts(ctx context.Context, locale *string) ([]models.BlogPost, error) {
	return s.collectBlogPosts(ctx, `SELECT `+blogColumns+` FROM blog_posts
		WHERE is_published AND ($1::text IS NULL OR locale = $1)
		ORDER BY published_at DESC, created_at DESC`, locale)
}

// AllBlogPosts lists drafts too, for the admin panel.
func (s *Store) AllBlogPosts(ctx context.Context, locale *string) ([]models.BlogPost, error) {
	return s.collectBlogPosts(ctx, `SELECT `+blogColumns+` FROM blog_posts
		WHERE $1::text IS NULL OR locale = $1
		ORDER BY updated_at DESC`, locale)
}

// BlogPostsByUser lists a partner's own posts, drafts included.
func (s *Store) BlogPostsByUser(ctx context.Context, userID string) ([]models.BlogPost, error) {
	return s.collectBlogPosts(ctx, `SELECT `+blogColumns+` FROM blog_posts
		WHERE user_id = $1 ORDER BY created_at DESC`, userID)
}

func (s *Store) BlogPost(ctx context.Context, id string) (models.BlogPost, error) {
	p, err := scanBlogPost(s.db.QueryRow(ctx, `SELECT `+blogColumns+` FROM blog_posts WHERE id = $1`, id))
	return p, notFound(err)
}

func (s *Store) PublishedBlogPostBySlug(ctx context.Context, locale, slug string) (models.BlogPost, error) {
	p, err := scanBlogPost(s.db.QueryRow(ctx, `SELECT `+blogColumns+` FROM blog_posts
		WHERE locale = $1 AND slug = $2 AND is_published`, locale, slug))
	return p, notFound(err)
}

// PublishedBlogLocales lists the locales a published post with slug is in,
// in the site's order: posts under the same slug are translations of one
// article.
func (s *Store) PublishedBlogLocales(ctx context.Context, slug string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT locale FROM blog_posts WHERE slug = $1 AND is_published
		ORDER BY array_position(ARRAY['ru', 'ky', 'en', 'de'], locale)`, slug)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) BlogSlugTaken(ctx context.Context, locale, slug, excludeID string) (bool, error) {
	return s.slugTaken(ctx, "blog_posts", locale, slug, excludeID)
}

func (s *Store) InsertBlogPost(ctx context.Context, p models.BlogPost) (models.BlogPost, error) {
	saved, err := scanBlogPost(s.db.QueryRow(ctx, `INSERT INTO blog_posts
		(locale, slug, title, excerpt, content, cover_image_url, seo_title, seo_description, difficulty,
		 duration, season, is_published, published_at, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING `+blogColumns,
		p.Locale, p.Slug, p.Title, p.Excerpt, p.Content, p.CoverImageURL, p.SeoTitle, p.SeoDescription,
		p.Difficulty, p.Duration, p.Season, p.IsPublished, publishedTime(p.PublishedAt), p.UserID))
	return saved, uniqueViolation(err)
}

func (s *Store) UpdateBlogPost(ctx context.Context, p models.BlogPost) (models.BlogPost, error) {
	saved, err := scanBlogPost(s.db.QueryRow(ctx, `UPDATE blog_posts SET
		locale = $2, slug = $3, title = $4, excerpt = $5, content = $6, cover_image_url = $7,
		seo_title = $8, seo_description = $9, difficulty = $10, duration = $11, season = $12,
		is_published = $13, published_at = $14, updated_at = now()
		WHERE id = $1
		RETURNING `+blogColumns,
		p.ID, p.Locale, p.Slug, p.Title, p.Excerpt, p.Content, p.CoverImageURL, p.SeoTitle, p.SeoDescription,
		p.Difficulty, p.Duration, p.Season, p.IsPublished, publishedTime(p.PublishedAt)))
	return saved, uniqueViolation(notFound(err))
}

func (s *Store) DeleteBlogPost(ctx context.Context, id string) error {
	return s.deleteByID(ctx, "blog_posts", id)
}

// slugTaken and deleteByID take a table name from a fixed set of callers,
// never from input.
func (s *Store) slugTaken(ctx context.Context, table, locale, slug, excludeID string) (bool, error) {
	var taken bool
	err := s.db.QueryRow(ctx, fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s
		WHERE locale = $1 AND slug = $2 AND ($3 = '' OR id::text <> $3))`, table), locale, slug, excludeID).Scan(&taken)
	return taken, err
}

func (s *Store) deleteByID(ctx context.Context, table, id string) error {
	tag, err := s.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, table), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func publishedTime(t *jsontime.Time) *time.Time {
	if t == nil {
		return nil
	}
	return &t.Time
}
