// Package mapdata holds the boards this app plays, copied from the monorepo-shared /map/boards by `make map` — this
// app's half of that source, the way generated/proto is its half of /proto. A board is a directory: its manifest
// (board-<hash>.json) and the files the manifest names.
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
	"path"
)

//go:embed boards
var files embed.FS

// Boards names the boards this app carries.
func Boards() ([]string, error) {
	entries, err := fs.ReadDir(files, "boards")
	if err != nil {
		return nil, fmt.Errorf("failed to list the boards: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}

// Manifest returns a board's manifest and the name it is content-addressed under.
func Manifest(board string) ([]byte, string, error) {
	// Globbed: the name is content-addressed, so a stale copy shows in it.
	entries, err := fs.Glob(files, path.Join("boards", board, "board-*.json"))
	if err != nil {
		return nil, "", fmt.Errorf("failed to glob board %s's manifest: %w", board, err)
	}
	if len(entries) != 1 {
		// Two means a `make map` that copied without sweeping the previous one.
		return nil, "", fmt.Errorf("expected exactly one manifest for board %s, found %d: %v", board, len(entries), entries)
	}

	blob, err := files.ReadFile(entries[0])
	if err != nil {
		return nil, "", fmt.Errorf("failed to read %s: %w", entries[0], err)
	}

	return blob, path.Base(entries[0]), nil
}

// File returns one of the files a board's manifest names.
func File(board, name string) ([]byte, error) {
	if path.Base(name) != name {
		return nil, fmt.Errorf("%q is not a file of board %s", name, board)
	}

	blob, err := files.ReadFile(path.Join("boards", board, name))
	if err != nil {
		return nil, fmt.Errorf("failed to read board %s's %s: %w", board, name, err)
	}

	return blob, nil
}
