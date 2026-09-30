// Package migrations is the activity schema, which the planet migrates while it builds.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
