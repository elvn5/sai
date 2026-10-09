// Package legacy reads the NestJS (TypeORM) database the services import
// their data from once.
//
// That database may be at any NestJS release: production, for instance,
// can still be at a schema from before columns like users."partnerStatus"
// existed. Select therefore builds queries that read each column if it
// exists and otherwise use the default the missing NestJS migration would
// have filled in, so an import works from any older schema.
package legacy

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Column is one value read from a legacy table.
type Column struct {
	Name string // TypeORM column name, e.g. "partnerStatus"
	// Cast is appended when the column exists, e.g. "::text" for enums.
	Cast string
	// Fallback is the SQL used when the column doesn't exist; empty means
	// the column is required (it existed in every NestJS release).
	Fallback string
}

// Col is a required column.
func Col(name string) Column { return Column{Name: name} }

// TableExists reports whether public.<table> exists.
func TableExists(ctx context.Context, conn *pgx.Conn, table string) (bool, error) {
	var exists bool
	err := conn.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists)
	return exists, err
}

// Select builds a SELECT over table reading cols in order.
func Select(ctx context.Context, conn *pgx.Conn, table string, cols []Column) (string, error) {
	rows, err := conn.Query(ctx, `SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1`, table)
	if err != nil {
		return "", err
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return "", err
	}
	present := make(map[string]bool, len(names))
	for _, n := range names {
		present[n] = true
	}

	exprs := make([]string, len(cols))
	for i, c := range cols {
		switch {
		case present[c.Name]:
			exprs[i] = pgx.Identifier{c.Name}.Sanitize() + c.Cast
		case c.Fallback != "":
			exprs[i] = c.Fallback
		default:
			return "", fmt.Errorf("legacy table %s has no column %q", table, c.Name)
		}
	}
	return "SELECT " + strings.Join(exprs, ", ") + " FROM " + pgx.Identifier{table}.Sanitize(), nil
}
