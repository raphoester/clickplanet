// Package migrations is the schema for every module, embedded so the binary migrates itself at boot.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
