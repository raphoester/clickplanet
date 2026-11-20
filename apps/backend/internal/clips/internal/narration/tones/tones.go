// Package tones holds the tones a war report is written in (docs/video-lab.md, "The war report's bench"), each a brief,
// embedded: <name>.md. A tone is a file, and a commit adds or changes one.
package tones

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
)

//go:embed *.md
var briefs embed.FS

// Every tone, by name.
func All() ([]narration.Tone, error) {
	files, err := fs.Glob(briefs, "*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to list the tones: %w", err)
	}
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, strings.TrimSuffix(file, ".md"))
	}
	return Named(slices.Sorted(slices.Values(names))...)
}

// The tones named, in that order: one with no brief is an error.
func Named(names ...string) ([]narration.Tone, error) {
	tones := make([]narration.Tone, 0, len(names))
	for _, name := range names {
		brief, err := briefs.ReadFile(name + ".md")
		if err != nil {
			return nil, fmt.Errorf("no tone is named %q: %w", name, err)
		}
		tones = append(tones, narration.Tone{Name: narration.ToneName(name), Brief: string(brief)})
	}
	return tones, nil
}
