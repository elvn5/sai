// Package legacyimport copies content out of the NestJS API's database
// into the content service's own database, once.
//
// It runs before the service starts serving: on the first start after the
// gateway switches content traffic here, the existing blog posts, guides,
// gallery items and page SEO are copied with their ids and timestamps, and
// a row in legacy_imports records that it's done. It never runs again, so
// later edits here are never overwritten. The NestJS tables are only read.
package legacyimport

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/pkg/legacy"
)

const importName = "nestjs-content"

// Run imports from the legacy database at legacyDSN unless already done.
func Run(ctx context.Context, db *pgxpool.Pool, legacyDSN string, log *slog.Logger) error {
	var done bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM legacy_imports WHERE name = $1)`, importName).Scan(&done); err != nil {
		return fmt.Errorf("check import marker: %w", err)
	}
	if done {
		return nil
	}

	conn, ok, err := legacy.Connect(ctx, legacyDSN)
	if err != nil {
		return err
	}
	if !ok {
		log.Info("no legacy database; nothing to import")
		return nil
	}
	defer func() { _ = conn.Close(ctx) }()

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, step := range steps {
		n, err := step.run(ctx, conn, tx)
		if err != nil {
			return fmt.Errorf("import %s: %w", step.table, err)
		}
		log.Info("imported legacy rows", slog.String("table", step.table), slog.Int("rows", n))
	}

	if _, err := tx.Exec(ctx, `INSERT INTO legacy_imports (name) VALUES ($1)`, importName); err != nil {
		return fmt.Errorf("record import: %w", err)
	}
	return tx.Commit(ctx)
}

type step struct {
	table  string
	cols   []legacy.Column // read from the legacy table, in insert order
	insert string          // into the content database; ON CONFLICT DO NOTHING
	// convert adapts a legacy row to the insert's parameters.
	convert func(values []any) []any
}

var steps = []step{
	{
		table: "blog_posts",
		cols: []legacy.Column{
			legacy.Col("id"), {Name: "locale", Cast: "::text"}, legacy.Col("slug"), legacy.Col("title"),
			legacy.Col("excerpt"), legacy.Col("content"), legacy.Col("coverImageUrl"), legacy.Col("seoTitle"),
			legacy.Col("seoDescription"), legacy.Col("isPublished"), legacy.Col("publishedAt"),
			legacy.Col("createdAt"), legacy.Col("updatedAt"),
		},
		insert: `INSERT INTO blog_posts (id, locale, slug, title, excerpt, content, cover_image_url,
			seo_title, seo_description, is_published, published_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT DO NOTHING`,
	},
	{
		table: "guides",
		cols: []legacy.Column{
			legacy.Col("id"), {Name: "locale", Cast: "::text"}, legacy.Col("slug"), legacy.Col("title"),
			legacy.Col("excerpt"), legacy.Col("content"), {Name: "difficulty", Cast: "::text"},
			{Name: "season", Cast: "::text"}, legacy.Col("equipmentChecklist"), legacy.Col("coverImageUrl"),
			{Name: "attachments", Fallback: "'[]'"}, legacy.Col("seoTitle"), legacy.Col("seoDescription"),
			legacy.Col("isPublished"), legacy.Col("publishedAt"), {Name: "userId", Fallback: "NULL::uuid"},
			legacy.Col("createdAt"), legacy.Col("updatedAt"),
		},
		insert: `INSERT INTO guides (id, locale, slug, title, excerpt, content, difficulty, season,
			equipment_checklist, cover_image_url, attachments, seo_title, seo_description, is_published,
			published_at, user_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12, $13, $14, $15, $16, $17, $18)
			ON CONFLICT DO NOTHING`,
		convert: func(v []any) []any {
			// TypeORM simple-array: comma-joined text, "" for empty.
			list, _ := v[8].(string)
			items := []string{}
			if list != "" {
				items = strings.Split(list, ",")
			}
			v[8] = items
			// TypeORM simple-json: JSON text.
			if s, ok := v[10].(string); !ok || s == "" {
				v[10] = "[]"
			}
			return v
		},
	},
	{
		table: "gallery_items",
		cols: []legacy.Column{
			legacy.Col("id"), legacy.Col("imageUrl"),
			// Older releases had a "region" enum instead; the NestJS migration
			// that replaced it dropped those values, so they're not carried over.
			{Name: "location", Fallback: "NULL"},
			legacy.Col("caption"), legacy.Col("altText"), legacy.Col("sortOrder"), legacy.Col("isPublished"),
			{Name: "userId", Fallback: "NULL::uuid"}, legacy.Col("createdAt"), legacy.Col("updatedAt"),
		},
		insert: `INSERT INTO gallery_items (id, image_url, location, caption, alt_text, sort_order,
			is_published, user_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT DO NOTHING`,
	},
	{
		table: "page_seo",
		cols: []legacy.Column{
			legacy.Col("id"), {Name: "page", Cast: "::text"}, {Name: "locale", Cast: "::text"},
			legacy.Col("title"), legacy.Col("description"), legacy.Col("ogImageUrl"), legacy.Col("updatedAt"),
		},
		insert: `INSERT INTO page_seo (id, page, locale, title, description, og_image_url, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`,
	},
}

func (s step) run(ctx context.Context, conn *pgx.Conn, tx pgx.Tx) (int, error) {
	// A fresh install may not have the table; nothing to copy then.
	exists, err := legacy.TableExists(ctx, conn, s.table)
	if err != nil || !exists {
		return 0, err
	}
	query, err := legacy.Select(ctx, conn, s.table, s.cols)
	if err != nil {
		return 0, err
	}
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return 0, err
	}
	var all [][]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			rows.Close()
			return 0, err
		}
		if s.convert != nil {
			values = s.convert(values)
		}
		all = append(all, values)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, values := range all {
		if _, err := tx.Exec(ctx, s.insert, values...); err != nil {
			return 0, err
		}
	}
	return len(all), nil
}
