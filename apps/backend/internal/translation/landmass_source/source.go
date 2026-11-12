// Package landmass_source reads the names of the landmasses that are not a country's main one, as the page's
// static/landmassNames.json holds them, each with the names Natural Earth gives it in other languages.
package landmass_source

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/catalogue_files"
)

type Gazetteer interface {
	// Natural Earth's language suffix ("fr", "pt") → its name for the place.
	NamesOf(english string) map[string]string
}

const kind = "A place on the world map: an island, a region or a province."

func Read(path string, gazetteer Gazetteer) ([]translation.Line, []translation.Key, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a path the operator gave.
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var file struct {
		Names json.RawMessage `json:"names"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, nil, fmt.Errorf("%s does not read: %w", path, err)
	}
	names, err := catalogue_files.Parse(file.Names)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	lines := make([]translation.Line, 0, len(names.Order))
	for _, key := range names.Order {
		english := names.Lines[key]
		lines = append(lines, translation.Line{Key: key, English: english, Context: contextOf(english, gazetteer)})
	}
	return lines, names.Order, nil
}

func contextOf(english string, gazetteer Gazetteer) string {
	base, number := split(english)
	context := kind
	if number != "" {
		context += fmt.Sprintf(" %q tells it from another place named %q: keep it.", number, base)
	}
	known := gazetteer.NamesOf(base)
	if len(known) == 0 {
		return context
	}
	languages := make([]string, 0, len(known))
	for language := range known {
		languages = append(languages, language)
	}
	slices.Sort(languages)
	named := make([]string, 0, len(languages))
	for _, language := range languages {
		named = append(named, fmt.Sprintf("%s %q", language, known[language]))
	}
	return context + " Natural Earth names it: " + strings.Join(named, ", ") + "."
}

// "Ionian Islands 2" is the second of the name.
func split(english string) (string, string) {
	at := strings.LastIndexByte(english, ' ')
	if at < 0 {
		return english, ""
	}
	tail := english[at+1:]
	if tail == "" || strings.Trim(tail, "0123456789") != "" {
		return english, ""
	}
	return english[:at], " " + tail
}
