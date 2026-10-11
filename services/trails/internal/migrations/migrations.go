// Package migrations holds the trails database schema (goose SQL files).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
