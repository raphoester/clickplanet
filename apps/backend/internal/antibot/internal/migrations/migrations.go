// Package migrations is the antibot schema, embedded so the guard migrates it while the planet builds it.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
