package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/content/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

const guideColumns = `id, locale, slug, title, excerpt, content, difficulty, season,
	equipment_checklist, cover_image_url, attachments, seo_title, seo_description, is_published,
	published_at, user_id::text, created_at, updated_at`

func scanGuide(row pgx.Row) (models.Guide, error) {
	var g models.Guide
	var publishedAt *time.Time
	var createdAt, updatedAt time.Time
	err := row.Scan(&g.ID, &g.Locale, &g.Slug, &g.Title, &g.Excerpt, &g.Content, &g.Difficulty, &g.Season,
		&g.EquipmentChecklist, &g.CoverImageURL, &g.Attachments, &g.SeoTitle, &g.SeoDescription,
		&g.IsPublished, &publishedAt, &g.UserID, &createdAt, &updatedAt)
	g.PublishedAt, g.CreatedAt, g.UpdatedAt = timePtr(publishedAt), jsontime.New(createdAt), jsontime.New(updatedAt)
	if g.EquipmentChecklist == nil {
		g.EquipmentChecklist = []string{}
	}
	if g.Attachments == nil {
		g.Attachments = []models.Attachment{}
	}
	return g, err
}

func (s *Store) collectGuides(ctx context.Context, query string, args ...any) ([]models.Guide, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Guide, error) { return scanGuide(row) })
}

func (s *Store) PublishedGuides(ctx context.Context, locale *string) ([]models.Guide, error) {
	return s.collectGuides(ctx, `SELECT `+guideColumns+` FROM guides
		WHERE is_published AND ($1::text IS NULL OR locale = $1)
		ORDER BY published_at DESC, created_at DESC`, locale)
}

func (s *Store) GuidesByUser(ctx context.Context, userID string) ([]models.Guide, error) {
	return s.collectGuides(ctx, `SELECT `+guideColumns+` FROM guides
		WHERE user_id = $1 ORDER BY created_at DESC`, userID)
}

func (s *Store) Guide(ctx context.Context, id string) (models.Guide, error) {
	g, err := scanGuide(s.db.QueryRow(ctx, `SELECT `+guideColumns+` FROM guides WHERE id = $1`, id))
	return g, notFound(err)
}

func (s *Store) PublishedGuideBySlug(ctx context.Context, locale, slug string) (models.Guide, error) {
	g, err := scanGuide(s.db.QueryRow(ctx, `SELECT `+guideColumns+` FROM guides
		WHERE locale = $1 AND slug = $2 AND is_published`, locale, slug))
	return g, notFound(err)
}

func (s *Store) GuideSlugTaken(ctx context.Context, locale, slug, excludeID string) (bool, error) {
	return s.slugTaken(ctx, "guides", locale, slug, excludeID)
}

func (s *Store) InsertGuide(ctx context.Context, g models.Guide) (models.Guide, error) {
	saved, err := scanGuide(s.db.QueryRow(ctx, `INSERT INTO guides
		(locale, slug, title, excerpt, content, difficulty, season, equipment_checklist, cover_image_url,
		 attachments, seo_title, seo_description, is_published, published_at, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING `+guideColumns,
		g.Locale, g.Slug, g.Title, g.Excerpt, g.Content, g.Difficulty, g.Season, g.EquipmentChecklist,
		g.CoverImageURL, g.Attachments, g.SeoTitle, g.SeoDescription, g.IsPublished,
		publishedTime(g.PublishedAt), g.UserID))
	return saved, uniqueViolation(err)
}

func (s *Store) UpdateGuide(ctx context.Context, g models.Guide) (models.Guide, error) {
	saved, err := scanGuide(s.db.QueryRow(ctx, `UPDATE guides SET
		locale = $2, slug = $3, title = $4, excerpt = $5, content = $6, difficulty = $7, season = $8,
		equipment_checklist = $9, cover_image_url = $10, attachments = $11, updated_at = now()
		WHERE id = $1
		RETURNING `+guideColumns,
		g.ID, g.Locale, g.Slug, g.Title, g.Excerpt, g.Content, g.Difficulty, g.Season,
		g.EquipmentChecklist, g.CoverImageURL, g.Attachments))
	return saved, uniqueViolation(notFound(err))
}

func (s *Store) DeleteGuide(ctx context.Context, id string) error {
	return s.deleteByID(ctx, "guides", id)
}
