// Package page_source reads the game page's English lines, each with what a translator needs: the note written for
// it, and the lines of code that show it.
package page_source

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/catalogue_files"
)

type Files struct {
	English string
	// Key → a note for the translator, for a line whose code does not say what it means.
	Notes string
	// The page's code, searched for each key.
	Code string
}

// A key is seldom shown in more places than this, and the first say enough.
const usesShown = 3

const longestUse = 240

func Read(files Files) ([]translation.Line, []translation.Key, error) {
	english, err := catalogue_files.Read(files.English)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the English: %w", err)
	}
	notes, err := catalogue_files.Read(files.Notes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the notes: %w", err)
	}
	uses, err := usesOf(files.Code, english.Order)
	if err != nil {
		return nil, nil, err
	}
	lines := make([]translation.Line, 0, len(english.Order))
	for _, key := range english.Order {
		var context []string
		if note := notes.Lines[key]; note != "" {
			context = append(context, "Note: "+note)
		}
		context = append(context, uses[key]...)
		lines = append(lines, translation.Line{Key: key, English: english.Lines[key], Context: strings.Join(context, "\n")})
	}
	return lines, english.Order, nil
}

// Each key's first uses in the page's code, as "file: line", tests and the catalogue left out.
func usesOf(root string, keys []translation.Key) (map[translation.Key][]string, error) {
	uses := map[translation.Key][]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			if name == "gen" || name == "i18n" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".tsx") || strings.Contains(name, ".test.") {
			return nil
		}
		code, err := os.ReadFile(path) //nolint:gosec // the page's own code.
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		relative, _ := filepath.Rel(root, path)
		for _, text := range strings.Split(string(code), "\n") {
			for _, key := range keys {
				if len(uses[key]) >= usesShown || !strings.Contains(text, `"`+string(key)+`"`) {
					continue
				}
				uses[key] = append(uses[key], relative+": "+shortened(strings.TrimSpace(text)))
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search %s: %w", root, err)
	}
	return uses, nil
}

func shortened(text string) string {
	if len(text) <= longestUse {
		return text
	}
	return text[:longestUse] + "…"
}
