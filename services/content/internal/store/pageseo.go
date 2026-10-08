package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/content/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

const pageSeoColumns = `id, page, locale, title, description, og_image_url, updated_at`

func scanPageSeo(row pgx.Row) (models.PageSeo, error) {
	var p models.PageSeo
	var updatedAt time.Time
	err := row.Scan(&p.ID, &p.Page, &p.Locale, &p.Title, &p.Description, &p.OgImageURL, &updatedAt)
	p.UpdatedAt = jsontime.New(updatedAt)
	return p, err
}

func (s *Store) PageSeo(ctx context.Context, page, locale string) (models.PageSeo, error) {
	p, err := scanPageSeo(s.db.QueryRow(ctx, `SELECT `+pageSeoColumns+` FROM page_seo
		WHERE page = $1 AND locale = $2`, page, locale))
	return p, notFound(err)
}

func (s *Store) AllPageSeo(ctx context.Context) ([]models.PageSeo, error) {
	rows, err := s.db.Query(ctx, `SELECT `+pageSeoColumns+` FROM page_seo ORDER BY page, locale`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.PageSeo, error) { return scanPageSeo(row) })
}

func (s *Store) UpsertPageSeo(ctx context.Context, p models.PageSeo) (models.PageSeo, error) {
	return scanPageSeo(s.db.QueryRow(ctx, `INSERT INTO page_seo (page, locale, title, description, og_image_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (page, locale) DO UPDATE SET
			title = EXCLUDED.title, description = EXCLUDED.description,
			og_image_url = EXCLUDED.og_image_url, updated_at = now()
		RETURNING `+pageSeoColumns,
		p.Page, p.Locale, p.Title, p.Description, p.OgImageURL))
}
