package store

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/content/internal/models"
	"github.com/elvn5/sai/services/pkg/jsontime"
)

const galleryColumns = `id, image_url, location, caption, alt_text, sort_order, is_published,
	user_id::text, created_at, updated_at`

func scanGalleryItem(row pgx.Row) (models.GalleryItem, error) {
	var g models.GalleryItem
	var createdAt, updatedAt time.Time
	err := row.Scan(&g.ID, &g.ImageURL, &g.Location, &g.Caption, &g.AltText, &g.SortOrder,
		&g.IsPublished, &g.UserID, &createdAt, &updatedAt)
	g.CreatedAt, g.UpdatedAt = jsontime.New(createdAt), jsontime.New(updatedAt)
	return g, err
}

func (s *Store) collectGalleryItems(ctx context.Context, query string, args ...any) ([]models.GalleryItem, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.GalleryItem, error) { return scanGalleryItem(row) })
}

type GalleryFilter struct {
	Location string
	// Matched against caption and location with ILIKE. As in the NestJS
	// API, % and _ in the term act as wildcards.
	Search string
	Limit  int
	Offset int
}

func (s *Store) PublishedGalleryItems(ctx context.Context, f GalleryFilter) ([]models.GalleryItem, error) {
	var where strings.Builder
	where.WriteString("is_published")
	args := []any{}
	if f.Location != "" {
		args = append(args, f.Location)
		where.WriteString(" AND location = $" + strconv.Itoa(len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		n := strconv.Itoa(len(args))
		where.WriteString(" AND (caption ILIKE $" + n + " OR location ILIKE $" + n + ")")
	}
	args = append(args, f.Limit, f.Offset)
	return s.collectGalleryItems(ctx, `SELECT `+galleryColumns+` FROM gallery_items
		WHERE `+where.String()+`
		ORDER BY sort_order ASC, created_at DESC
		LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
}

func (s *Store) GalleryLocations(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT DISTINCT location FROM gallery_items
		WHERE is_published AND location IS NOT NULL ORDER BY location ASC`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) GalleryItemsByUser(ctx context.Context, userID string) ([]models.GalleryItem, error) {
	return s.collectGalleryItems(ctx, `SELECT `+galleryColumns+` FROM gallery_items
		WHERE user_id = $1 ORDER BY created_at DESC`, userID)
}

func (s *Store) GalleryItem(ctx context.Context, id string) (models.GalleryItem, error) {
	g, err := scanGalleryItem(s.db.QueryRow(ctx, `SELECT `+galleryColumns+` FROM gallery_items WHERE id = $1`, id))
	return g, notFound(err)
}

func (s *Store) InsertGalleryItem(ctx context.Context, g models.GalleryItem) (models.GalleryItem, error) {
	return scanGalleryItem(s.db.QueryRow(ctx, `INSERT INTO gallery_items
		(image_url, location, caption, alt_text, sort_order, is_published, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+galleryColumns,
		g.ImageURL, g.Location, g.Caption, g.AltText, g.SortOrder, g.IsPublished, g.UserID))
}

func (s *Store) UpdateGalleryItem(ctx context.Context, g models.GalleryItem) (models.GalleryItem, error) {
	saved, err := scanGalleryItem(s.db.QueryRow(ctx, `UPDATE gallery_items SET
		image_url = $2, location = $3, caption = $4, alt_text = $5, updated_at = now()
		WHERE id = $1
		RETURNING `+galleryColumns,
		g.ID, g.ImageURL, g.Location, g.Caption, g.AltText))
	return saved, notFound(err)
}

func (s *Store) DeleteGalleryItem(ctx context.Context, id string) error {
	return s.deleteByID(ctx, "gallery_items", id)
}
