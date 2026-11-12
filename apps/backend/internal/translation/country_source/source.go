// Package country_source reads the page's flags, static/countries/countries.json, as the lines each language needs
// to name a flag's country inside a sentence: "in Argentina", "en Argentine", "au Brésil".
package country_source

import (
	"fmt"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation/catalogue_files"
)

const inContext = `How the language says "in" this country inside a sentence, as in "Take 5 tiles in Argentina": the words ` +
	`before the name, its article and the case it takes there ("en Argentine", "au Brésil", "aux États-Unis", ` +
	`"na Argentina", "in der Schweiz", "w Argentynie", "Arjantin'de"). {flag} is the flag's picture, right before ` +
	`the name. The country's ISO code is %q.`

// Each flag's "in" line, keyed "in.<code>", in the file's order.
func Read(path string) ([]translation.Line, []translation.Key, error) {
	data, err := os.ReadFile(path) //nolint:gosec // a path the operator gave.
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	names, err := catalogue_files.Parse(data)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	lines := make([]translation.Line, 0, len(names.Order))
	order := make([]translation.Key, 0, len(names.Order))
	for _, code := range names.Order {
		key := "in." + code
		lines = append(lines, translation.Line{Key: key, English: "in {flag} " + names.Lines[code], Context: fmt.Sprintf(inContext, code)})
		order = append(order, key)
	}
	return lines, order, nil
}
