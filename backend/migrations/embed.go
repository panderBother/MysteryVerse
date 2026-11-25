package migrations

import "embed"

// FS contains the versioned goose migrations used by the migrate command.
//
//go:embed *.sql
var FS embed.FS
