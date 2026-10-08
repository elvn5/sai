// Package migrations holds the listings database schema (goose SQL files).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
