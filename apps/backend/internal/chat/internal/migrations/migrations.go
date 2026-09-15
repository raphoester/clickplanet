// Package migrations is the chat schema, embedded so the module migrates it while it builds.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
