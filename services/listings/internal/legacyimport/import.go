// Package legacyimport copies listings out of the NestJS API's database
// into the listings service's own database, once — on the first start, before
// serving. The NestJS tables are only read. See the content service's
// legacyimport for the same pattern.
package legacyimport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/pkg/legacy"
)

const importName = "nestjs-listings"

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

	for _, s := range steps {
		n, err := s.run(ctx, conn, tx)
		if err != nil {
			return fmt.Errorf("import %s: %w", s.table, err)
		}
		log.Info("imported legacy rows", slog.String("table", s.table), slog.Int("rows", n))
	}
	if _, err := tx.Exec(ctx, `INSERT INTO legacy_imports (name) VALUES ($1)`, importName); err != nil {
		return fmt.Errorf("record import: %w", err)
	}
	return tx.Commit(ctx)
}

type step struct {
	table  string
	cols   []legacy.Column
	insert string
	// Columns (by index in query) holding TypeORM simple-array / simple-json
	// text, converted to string slices.
	simpleArrays, simpleJSON []int
}

var steps = []step{
	{
		table: "hotels",
		cols: cols(`id slug name type:text description destination address latitude longitude priceFrom
			stars amenities cancellationPolicy checkInTime checkOutTime coverImageUrl photos phone whatsapp
			telegram website ratingAverage reviewCount isPublished userId createdAt updatedAt`),
		insert: `INSERT INTO hotels (id, slug, name, type, description, destination, address, latitude,
			longitude, price_from, stars, amenities, cancellation_policy, check_in_time, check_out_time,
			cover_image_url, photos, phone, whatsapp, telegram, website, rating_average, review_count,
			is_published, user_id, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
			ON CONFLICT DO NOTHING`,
		simpleArrays: []int{11},
		simpleJSON:   []int{16},
	},
	{
		table: "local_guides",
		cols: cols(`id slug name headline bio avatarUrl coverImageUrl photos languages specializations
			destinations pricePerDay experienceYears hasVehicle phone whatsapp telegram ratingAverage
			reviewCount isPublished userId createdAt updatedAt`),
		insert: `INSERT INTO local_guides (id, slug, name, headline, bio, avatar_url, cover_image_url,
			photos, languages, specializations, destinations, price_per_day, experience_years,
			has_vehicle, phone, whatsapp, telegram, rating_average, review_count, is_published, user_id,
			created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
			ON CONFLICT DO NOTHING`,
		simpleArrays: []int{8, 9, 10},
		simpleJSON:   []int{7},
	},
	{
		table: "tours",
		cols: append(cols(`id locale:text slug title excerpt content price durationDays location`),
			// Added by later NestJS migrations than the one creating the table.
			legacy.Column{Name: "startDate", Fallback: `"createdAt"::date`},
			legacy.Column{Name: "endDate", Fallback: `"createdAt"::date`},
			legacy.Column{Name: "destination", Fallback: "NULL"},
			legacy.Column{Name: "cancellationPolicy", Fallback: "NULL"},
			legacy.Col("coverImageUrl"),
			legacy.Column{Name: "phone", Fallback: "NULL"},
			legacy.Column{Name: "whatsapp", Fallback: "NULL"},
			legacy.Column{Name: "telegram", Fallback: "NULL"},
			legacy.Column{Name: "ratingAverage", Fallback: "0"},
			legacy.Column{Name: "reviewCount", Fallback: "0"},
			legacy.Col("isPublished"), legacy.Col("publishedAt"), legacy.Col("userId"),
			legacy.Col("createdAt"), legacy.Col("updatedAt"),
		),
		insert: `INSERT INTO tours (id, locale, slug, title, excerpt, content, price, duration_days,
			location, start_date, end_date, destination, cancellation_policy, cover_image_url, phone,
			whatsapp, telegram, rating_average, review_count, is_published, published_at, user_id,
			created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)
			ON CONFLICT DO NOTHING`,
	},
	{
		table: "reviews",
		cols:  cols(`id targetType targetId rating text photos userId createdAt updatedAt`),
		insert: `INSERT INTO reviews (id, target_type, target_id, rating, text, photos, user_id, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
		simpleJSON: []int{5},
	},
	{
		table: "listing_posts",
		cols:  cols(`id targetType targetId text photos userId createdAt updatedAt`),
		insert: `INSERT INTO listing_posts (id, target_type, target_id, text, photos, user_id, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`,
		simpleJSON: []int{4},
	},
}

// cols lists required columns by name; "name:text" casts an enum to text.
func cols(spec string) []legacy.Column {
	var out []legacy.Column
	for _, f := range strings.Fields(spec) {
		name, cast, _ := strings.Cut(f, ":")
		c := legacy.Col(name)
		if cast != "" {
			c.Cast = "::" + cast
		}
		out = append(out, c)
	}
	return out
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
		for _, i := range s.simpleArrays {
			values[i] = splitSimpleArray(values[i])
		}
		for _, i := range s.simpleJSON {
			if values[i], err = parseSimpleJSON(values[i]); err != nil {
				rows.Close()
				return 0, err
			}
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

// TypeORM simple-array: comma-joined text, "" when empty.
func splitSimpleArray(v any) []string {
	s, _ := v.(string)
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

// TypeORM simple-json: a JSON string array stored as text.
func parseSimpleJSON(v any) ([]string, error) {
	s, _ := v.(string)
	out := []string{}
	if s == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("parse simple-json %q: %w", s, err)
	}
	return out, nil
}
