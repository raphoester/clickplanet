// Package mapdata holds the tile coordinates blob, copied from the monorepo-shared /map by
// `make map` — this app's half of that source, the way generated/proto is its half of /proto.
//
// The directory is `map` to mirror the source it comes from; the package cannot be, since map is
// a keyword.
//
// It is embedded rather than read from disk: cmd/api is a self-contained container with no startup
// dependencies, and a file it has to find at boot is a boot that can fail for a reason nothing in
// this repo controls.
package mapdata

import (
	"embed"
	"fmt"
	"io/fs"
)

// Globbed because the file is content-addressed, which keeps `make map` a copy with no generated
// constant beside it, and makes drift between the two apps visible in the name.
//
//go:embed coordinates-*.bin
var files embed.FS

// Coordinates returns the blob and the name it is content-addressed under.
func Coordinates() ([]byte, string, error) {
	entries, err := fs.Glob(files, "coordinates-*.bin")
	if err != nil {
		return nil, "", fmt.Errorf("failed to glob the coordinates blob: %w", err)
	}
	if len(entries) != 1 {
		// Two means a `make map` that copied without sweeping the previous one.
		return nil, "", fmt.Errorf("expected exactly one coordinates blob, found %d: %v", len(entries), entries)
	}

	blob, err := files.ReadFile(entries[0])
	if err != nil {
		return nil, "", fmt.Errorf("failed to read %s: %w", entries[0], err)
	}

	return blob, entries[0], nil
}
