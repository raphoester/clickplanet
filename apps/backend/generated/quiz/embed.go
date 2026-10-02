// Package quizdata holds the quiz bank, copied from the monorepo-shared /quiz by `make quiz` —
// this app's half of that source, the way generated/map is its half of /map.
//
// It is this app's half and **nobody else's**: the frontend has no copy on purpose, because the
// answers are in here. A bank served to the page is a bank anyone can fetch with the network tab
// open. See /quiz/README.md.
//
// Embedded rather than read from disk, for the reason generated/map is: cmd/api is a self-contained
// container, and a file it has to find at boot is a boot that can fail for a reason nothing in this
// repo controls.
package quizdata

import (
	"embed"
	"fmt"
	"io/fs"
)

// Globbed because the file is content-addressed, which keeps `make quiz` a copy with no generated
// constant beside it.
//
//go:embed bank-*.json
var files embed.FS

// Bank returns the bank and the name it is content-addressed under.
func Bank() ([]byte, string, error) {
	entries, err := fs.Glob(files, "bank-*.json")
	if err != nil {
		return nil, "", fmt.Errorf("failed to glob the quiz bank: %w", err)
	}
	if len(entries) != 1 {
		// Two means a `make quiz` that copied without sweeping the previous one.
		return nil, "", fmt.Errorf("expected exactly one quiz bank, found %d: %v", len(entries), entries)
	}

	blob, err := files.ReadFile(entries[0])
	if err != nil {
		return nil, "", fmt.Errorf("failed to read %s: %w", entries[0], err)
	}

	return blob, entries[0], nil
}
