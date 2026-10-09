// Package migrations holds the users database schema (goose SQL files).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
