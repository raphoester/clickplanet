// Package mapdata holds the boards this app plays, copied from the monorepo-shared /map/boards by `make map` — this
// app's half of that source, the way generated/proto is its half of /proto. A board is a directory: its manifest
// (board-<hash>.json) and the files the manifest names. internal/shared/cpboards reads them. The carry-overs between
// two boards come from /map/carries, which a switch of the world's board reads.
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
	"io/fs"
)

//go:embed boards carries
var files embed.FS

// FS is every board this app carries, a directory each.
func FS() fs.FS {
	return sub("boards")
}

// Carries are the carry-overs between two boards: carry-<from>-<to>-<hash>.bin.
func Carries() fs.FS {
	return sub("carries")
}

func sub(directory string) fs.FS {
	tree, err := fs.Sub(files, directory)
	if err != nil {
		panic(err) // a constant path, which fs.Sub refuses only when it is malformed.
	}
	return tree
}
