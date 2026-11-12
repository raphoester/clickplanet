// Package catalogue_files reads and writes a catalogue as the page holds it: one JSON object per language, its keys in
// the English file's order, and beside it the fingerprint of the English each line came from.
package catalogue_files

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
)

// One file of keyed lines, in its own order.
type Keyed struct {
	Order []translation.Key
	Lines map[translation.Key]string
}

var errNotAnObject = errors.New("not a JSON object of strings")

// Reads a JSON object of strings and keeps its keys' order. A file that is not there is empty.
func Read(path string) (Keyed, error) {
	keyed := Keyed{Lines: map[translation.Key]string{}}
	data, err := os.ReadFile(path) //nolint:gosec // a path the operator gave.
	if errors.Is(err, fs.ErrNotExist) {
		return keyed, nil
	}
	if err != nil {
		return Keyed{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	keyed, err = Parse(data)
	if err != nil {
		return Keyed{}, fmt.Errorf("%s: %w", path, err)
	}
	return keyed, nil
}

// Reads a JSON object of strings and keeps its keys' order.
func Parse(data []byte) (Keyed, error) {
	keyed := Keyed{Lines: map[translation.Key]string{}}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return Keyed{}, errNotAnObject
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return Keyed{}, fmt.Errorf("%w: %w", errNotAnObject, err)
		}
		key, ok := token.(string)
		if !ok {
			return Keyed{}, errNotAnObject
		}
		var value string
		if err := decoder.Decode(&value); err != nil {
			return Keyed{}, fmt.Errorf("%q: %w", key, errNotAnObject)
		}
		keyed.Order = append(keyed.Order, translation.Key(key))
		keyed.Lines[translation.Key(key)] = value
	}
	return keyed, nil
}

// Writes the lines in the given order, the HTML in them as it is, two spaces deep as the page's files are.
func Write(path string, order []translation.Key, lines map[translation.Key]string) error {
	var out bytes.Buffer
	out.WriteString("{")
	first := true
	for _, key := range order {
		text, ok := lines[key]
		if !ok {
			continue
		}
		if !first {
			out.WriteString(",")
		}
		first = false
		out.WriteString("\n  ")
		out.Write(quoted(string(key)))
		out.WriteString(": ")
		out.Write(quoted(text))
	}
	if !first {
		out.WriteString("\n")
	}
	out.WriteString("}\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("failed to make the folder of %s: %w", path, err)
	}
	if err := os.WriteFile(path, out.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func quoted(text string) []byte {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(text) // A string always encodes.
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n"))
}

// A catalogue's place on disk: the English, each language's lines in a folder, their fingerprints in another.
type Place struct {
	English      string
	Lines        string
	Fingerprints string
}

func (p Place) linesOf(language translation.Language) string {
	return filepath.Join(p.Lines, string(language)+".json")
}

func (p Place) fingerprintsOf(language translation.Language) string {
	return filepath.Join(p.Fingerprints, string(language)+".json")
}

// The languages are the files of the lines' folder: a new language is an empty "{}" file named after its tag.
func (p Place) Languages() ([]translation.Language, error) {
	entries, err := os.ReadDir(p.Lines)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", p.Lines, err)
	}
	english, err := filepath.Abs(p.English)
	if err != nil {
		return nil, fmt.Errorf("failed to find %s: %w", p.English, err)
	}
	var languages []translation.Language
	for _, entry := range entries {
		path, err := filepath.Abs(filepath.Join(p.Lines, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to find %s: %w", entry.Name(), err)
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || path == english {
			continue
		}
		languages = append(languages, translation.Language(strings.TrimSuffix(entry.Name(), ".json")))
	}
	slices.Sort(languages)
	return languages, nil
}

func (p Place) Catalogue(language translation.Language) (translation.Catalogue, error) {
	lines, err := Read(p.linesOf(language))
	if err != nil {
		return translation.Catalogue{}, err
	}
	fingerprints, err := Read(p.fingerprintsOf(language))
	if err != nil {
		return translation.Catalogue{}, err
	}
	catalogue := translation.Catalogue{Lines: lines.Lines, Fingerprints: map[translation.Key]translation.Fingerprint{}}
	for key, fingerprint := range fingerprints.Lines {
		catalogue.Fingerprints[key] = translation.Fingerprint(fingerprint)
	}
	return catalogue, nil
}

func (p Place) Save(language translation.Language, order []translation.Key, catalogue translation.Catalogue) error {
	if err := Write(p.linesOf(language), order, catalogue.Lines); err != nil {
		return err
	}
	fingerprints := make(map[translation.Key]string, len(catalogue.Fingerprints))
	for key, fingerprint := range catalogue.Fingerprints {
		fingerprints[key] = string(fingerprint)
	}
	return Write(p.fingerprintsOf(language), order, fingerprints)
}
