// Package landmass_source reads the names of the landmasses that are not a country's main one, as the page serves them
// beside each board (static/boards/<board>/landmassNames-<hash>.json), each with the names Natural Earth gives it in
// other languages. A landmass id is a row of one board's borders file, so each board has names of its own.
package landmass_source

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// One board's landmass names, as the page serves them beside the board.
type Board struct {
	Name string
	Path string
}

var (
	namesFile = regexp.MustCompile(`^landmassNames-[0-9a-f]{8}\.json$`)

	ErrNamesTwice = errors.New("a board has more than one file of landmass names")
)

// Each board's names, by the board's name, in the page's copies of the boards (its static/boards). A board with none
// yet is left out: the page cannot play it until `npm run landmassNames` names it.
func Boards(copies string) ([]Board, error) {
	entries, err := os.ReadDir(copies)
	if err != nil {
		return nil, fmt.Errorf("failed to list the boards in %s: %w", copies, err)
	}
	var boards []Board
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(copies, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to list board %s: %w", entry.Name(), err)
		}
		var found []string
		for _, file := range files {
			if namesFile.MatchString(file.Name()) {
				found = append(found, file.Name())
			}
		}
		switch len(found) {
		case 0:
		case 1:
			boards = append(boards, Board{Name: entry.Name(), Path: filepath.Join(copies, entry.Name(), found[0])})
		default:
			return nil, fmt.Errorf("%w: %s holds %v", ErrNamesTwice, entry.Name(), found)
		}
	}
	return boards, nil
}

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
