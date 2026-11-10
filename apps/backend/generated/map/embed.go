// Package mapdata holds the tile coordinates, borders, capitals and waters blobs, copied from the monorepo-shared /map by
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
//go:embed coordinates-*.bin borders-*.bin capitals-*.json waters-*.json
var files embed.FS

// Coordinates returns the blob and the name it is content-addressed under.
func Coordinates() ([]byte, string, error) {
	return read("coordinates", "bin")
}

// Borders returns the tile to landmass table and the name it is content-addressed under.
func Borders() ([]byte, string, error) {
	return read("borders", "bin")
}

// Capitals returns each country's capital tile and the name it is content-addressed under.
func Capitals() ([]byte, string, error) {
	return read("capitals", "json")
}

// Waters returns each body of water's coast tiles and the name it is content-addressed under.
func Waters() ([]byte, string, error) {
	return read("waters", "json")
}

func read(asset, extension string) ([]byte, string, error) {
	entries, err := fs.Glob(files, asset+"-*."+extension)
	if err != nil {
		return nil, "", fmt.Errorf("failed to glob the %s blob: %w", asset, err)
	}
	if len(entries) != 1 {
		// Two means a `make map` that copied without sweeping the previous one.
		return nil, "", fmt.Errorf("expected exactly one %s blob, found %d: %v", asset, len(entries), entries)
	}

	blob, err := files.ReadFile(entries[0])
	if err != nil {
		return nil, "", fmt.Errorf("failed to read %s: %w", entries[0], err)
	}

	return blob, entries[0], nil
}
