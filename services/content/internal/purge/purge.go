// Package purge removes the guides and gallery photos that accounts other
// than approved partners published before publishing became partner-only
// (see services/pkg/partnerpurge).
package purge

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elvn5/sai/services/content/internal/handlers"
	"github.com/elvn5/sai/services/pkg/partnerpurge"
)

// Service describes the content tables to partnerpurge. media (optional)
// deletes the removed gallery photos from storage.
func Service(media handlers.PhotoDeleter, log *slog.Logger) partnerpurge.Service {
	var photos []string
	return partnerpurge.Service{
		Owners: func(ctx context.Context, db *pgxpool.Pool) ([]string, error) {
			rows, err := db.Query(ctx, `SELECT user_id::text FROM guides WHERE user_id IS NOT NULL
				UNION SELECT user_id::text FROM gallery_items WHERE user_id IS NOT NULL`)
			if err != nil {
				return nil, err
			}
			return pgx.CollectRows(rows, pgx.RowTo[string])
		},
		Purge: func(ctx context.Context, tx pgx.Tx, userIDs []string) (map[string]int64, error) {
			guides, err := tx.Exec(ctx, `DELETE FROM guides WHERE user_id::text = ANY($1)`, userIDs)
			if err != nil {
				return nil, err
			}
			rows, err := tx.Query(ctx, `DELETE FROM gallery_items WHERE user_id::text = ANY($1) RETURNING image_url`, userIDs)
			if err != nil {
				return nil, err
			}
			if photos, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
				return nil, err
			}
			return map[string]int64{"guides": guides.RowsAffected(), "galleryItems": int64(len(photos))}, nil
		},
		AfterCommit: func(ctx context.Context) {
			if media == nil {
				return
			}
			for _, url := range photos {
				if err := media.DeleteObject(ctx, url); err != nil {
					log.ErrorContext(ctx, "failed to delete gallery photo from storage",
						slog.String("imageUrl", url), slog.Any("error", err))
				}
			}
		},
	}
}
