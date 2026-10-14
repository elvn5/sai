package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/elvn5/sai/services/listings/internal/models"
)

// ownedListingsSQL selects the listings of user $1 as
// (target_type, id, title, slug, locale, is_published).
const ownedListingsSQL = `SELECT 'hotel' AS target_type, id, name AS title, slug, NULL::text AS locale, is_published
		FROM hotels WHERE user_id = $1
	UNION ALL SELECT 'local_guide', id, name, slug, NULL, is_published FROM local_guides WHERE user_id = $1
	UNION ALL SELECT 'tour', id, title, slug, locale, is_published FROM tours WHERE user_id = $1`

// statColumns maps a stats event to its counter. Values are fixed, never
// taken from input.
var statColumns = map[string]string{
	models.EventView:     "views",
	models.EventPhone:    "phone_clicks",
	models.EventWhatsapp: "whatsapp_clicks",
	models.EventTelegram: "telegram_clicks",
	models.EventWebsite:  "website_clicks",
}

// RecordEvent counts one stats event of a listing on a day.
func (s *Store) RecordEvent(ctx context.Context, targetType, targetID, event string, day time.Time) error {
	column := statColumns[event]
	_, err := s.db.Exec(ctx, `INSERT INTO listing_stats_daily (target_type, target_id, day, `+column+`)
		VALUES ($1, $2, $3::date, 1)
		ON CONFLICT (target_type, target_id, day) DO UPDATE SET `+column+` = listing_stats_daily.`+column+` + 1`,
		targetType, targetID, day.Format(time.DateOnly))
	return err
}

// OwnedListings lists a user's listings by kind and title.
func (s *Store) OwnedListings(ctx context.Context, userID string) ([]models.ListingRef, error) {
	rows, err := s.db.Query(ctx, `SELECT target_type, id::text, title, slug, locale, is_published
		FROM (`+ownedListingsSQL+`) l ORDER BY target_type, title`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (models.ListingRef, error) {
		var l models.ListingRef
		err := r.Scan(&l.TargetType, &l.ID, &l.Title, &l.Slug, &l.Locale, &l.IsPublished)
		return l, err
	})
}

const statSums = `COALESCE(sum(s.views), 0)::int, COALESCE(sum(s.phone_clicks), 0)::int,
	COALESCE(sum(s.whatsapp_clicks), 0)::int, COALESCE(sum(s.telegram_clicks), 0)::int,
	COALESCE(sum(s.website_clicks), 0)::int`

const ownedStatsFrom = `FROM listing_stats_daily s JOIN (` + ownedListingsSQL + `) l
	ON l.target_type = s.target_type AND l.id = s.target_id
	WHERE s.day BETWEEN $2::date AND $3::date`

func scanCounts(row pgx.Row, extra ...any) (models.StatCounts, error) {
	var c models.StatCounts
	err := row.Scan(append(extra, &c.Views, &c.Phone, &c.Whatsapp, &c.Telegram, &c.Website)...)
	c.Contacts = c.Phone + c.Whatsapp + c.Telegram + c.Website
	return c, err
}

// OwnerTotals sums the stats of a user's listings over [from, to].
func (s *Store) OwnerTotals(ctx context.Context, userID string, from, to time.Time) (models.StatCounts, error) {
	return scanCounts(s.db.QueryRow(ctx, `SELECT `+statSums+` `+ownedStatsFrom,
		userID, from.Format(time.DateOnly), to.Format(time.DateOnly)))
}

// OwnerStatsByListing sums the stats of each of a user's listings over
// [from, to], keyed by target type and id. Listings without any are left out.
func (s *Store) OwnerStatsByListing(ctx context.Context, userID string, from, to time.Time) (map[[2]string]models.StatCounts, error) {
	rows, err := s.db.Query(ctx, `SELECT s.target_type, s.target_id::text, `+statSums+` `+ownedStatsFrom+`
		GROUP BY s.target_type, s.target_id`, userID, from.Format(time.DateOnly), to.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[[2]string]models.StatCounts{}
	for rows.Next() {
		var key [2]string
		counts, err := scanCounts(rows, &key[0], &key[1])
		if err != nil {
			return nil, err
		}
		out[key] = counts
	}
	return out, rows.Err()
}

// OwnerDaily sums the stats of a user's listings per day over [from, to],
// keyed YYYY-MM-DD. Days without any are left out.
func (s *Store) OwnerDaily(ctx context.Context, userID string, from, to time.Time) (map[string]models.StatCounts, error) {
	rows, err := s.db.Query(ctx, `SELECT to_char(s.day, 'YYYY-MM-DD'), `+statSums+` `+ownedStatsFrom+`
		GROUP BY s.day`, userID, from.Format(time.DateOnly), to.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]models.StatCounts{}
	for rows.Next() {
		var day string
		counts, err := scanCounts(rows, &day)
		if err != nil {
			return nil, err
		}
		out[day] = counts
	}
	return out, rows.Err()
}
