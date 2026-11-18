// Package place_source reads the names of a board's places as the page serves them beside the board: its landmasses
// that are not a country's main one (static/boards/<board>/landmassNames-<hash>.json) and its provinces
// (provinceNames-<hash>.json), each with the names Natural Earth gives it in other languages. A landmass id is a row of
// one board's borders file, a province id of its provinces file, so each board has names of its own, but a name one
// board translates is the same on another.
package place_source

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

// Kind is the places a file names, by the start of its name.
type Kind string

const (
	Landmasses Kind = "landmassNames"
	Provinces  Kind = "provinceNames"
)

// One board's names of one kind of place, as the page serves them beside the board.
type Board struct {
	Name string
	Path string
}

var ErrNamesTwice = errors.New("a board has more than one file of names of a kind")

// Each board's names of a kind, by the board's name, in the page's copies of the boards (its static/boards). A board
// with none is left out: earth-1 has no provinces, and a board's landmasses are named by `npm run landmassNames`.
func Boards(copies string, kind Kind) ([]Board, error) {
	namesFile := regexp.MustCompile(`^` + string(kind) + `-[0-9a-f]{8}\.json$`)
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
	var context []string
	if number != "" {
		context = append(context, fmt.Sprintf("%q tells it from another place named %q: keep it.", strings.TrimSpace(number), base))
	}
	known := gazetteer.NamesOf(base)
	if len(known) == 0 {
		return strings.Join(context, " ")
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
	return strings.Join(append(context, "Natural Earth names it: "+strings.Join(named, ", ")+"."), " ")
}

// A language's names for the places of every board, by their English: a board takes the name another gives the same
// English, or the same before a trailing number ("Ionian Islands 2" is "Îles Ioniennes 2").
type Names map[string]string

func (n Names) Recall(english string) (string, bool) {
	if name, ok := n[english]; ok {
		return name, true
	}
	base, number := split(english)
	name, ok := n[base]
	if number == "" || !ok {
		return "", false
	}
	return name + number, true
}

// Learns a board's names in a language, the ones translated from today's English; a name known first stays.
func (n Names) Learn(source []translation.Line, current translation.Catalogue) {
	for _, line := range source {
		name, held := current.Lines[line.Key]
		was, fingerprinted := current.Fingerprints[line.Key]
		_, known := n[line.English]
		if !held || known || fingerprinted && was != translation.FingerprintOf(line.English) {
			continue
		}
		n[line.English] = name
	}
}

// NaturalEarth is what Natural Earth names a place in one of its languages ("fr", "pt"), with the place's number kept:
// a province takes its unit's name in each language Natural Earth has (docs/provinces.md). A name of more words than
// the English is left to the translator: Natural Earth names as a gazetteer does, with the kind of division the English
// does not have ("İsfahan Eyaleti", "Gespanschaft Split-Dalmatien").
type NaturalEarth struct {
	Gazetteer Gazetteer
	Language  string
}

func (n NaturalEarth) Recall(english string) (string, bool) {
	base, number := split(english)
	name, ok := n.Gazetteer.NamesOf(base)[n.Language]
	if !ok || len(strings.Fields(name)) > len(strings.Fields(base)) {
		return "", false
	}
	return name + number, true
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
