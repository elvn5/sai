// Package purge removes the hotels and local-guide profiles that accounts
// other than approved partners published before publishing became
// partner-only (see services/pkg/partnerpurge), with their reviews and
// posts.
package purge

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/pkg/partnerpurge"
)

// Service describes the listings tables to partnerpurge.
func Service() partnerpurge.Service {
	return partnerpurge.Service{
		Owners: func(ctx context.Context, db *pgxpool.Pool) ([]string, error) {
			rows, err := db.Query(ctx, `SELECT user_id::text FROM hotels WHERE user_id IS NOT NULL
				UNION SELECT user_id::text FROM local_guides WHERE user_id IS NOT NULL`)
			if err != nil {
				return nil, err
			}
			return pgx.CollectRows(rows, pgx.RowTo[string])
		},
		Purge: func(ctx context.Context, tx pgx.Tx, userIDs []string) (map[string]int64, error) {
			removed := map[string]int64{}
			for _, target := range []struct{ kind, table, key string }{
				{"hotel", "hotels", "hotels"},
				{"local_guide", "local_guides", "localGuides"},
			} {
				// Reviews and posts reference listings polymorphically, so
				// there's no cascade.
				for _, dependent := range []string{"reviews", "listing_posts"} {
					tag, err := tx.Exec(ctx, `DELETE FROM `+dependent+` WHERE target_type = $1
						AND target_id IN (SELECT id FROM `+target.table+` WHERE user_id::text = ANY($2))`, target.kind, userIDs)
					if err != nil {
						return nil, err
					}
					removed[dependent] += tag.RowsAffected()
				}
				tag, err := tx.Exec(ctx, `DELETE FROM `+target.table+` WHERE user_id::text = ANY($1)`, userIDs)
				if err != nil {
					return nil, err
				}
				removed[target.key] = tag.RowsAffected()
			}
			return removed, nil
		},
	}
}
