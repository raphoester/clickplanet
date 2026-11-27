// Package natural_earth reads the names Natural Earth gives a place in other languages, from the layers the page's
// landmass names were made of, as the page's scripts keep them.
package natural_earth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// The landmass names were made from this release (the page's scripts/map/naturalEarth.mjs).
const Tag = "v5.1.2"

type layer struct {
	name string
	// The admin-1 layer names its fields in lower case, the others in upper case.
	lower bool
}

var layers = []layer{
	{name: "ne_10m_geography_regions_polys"},
	{name: "ne_10m_geography_regions_points"},
	{name: "ne_10m_admin_0_map_subunits"},
	{name: "ne_50m_admin_0_map_subunits"},
	{name: "ne_10m_admin_1_states_provinces", lower: true},
}

// Natural Earth's languages, by the suffix of their name fields.
var languages = []string{"fr", "de", "es", "pt", "pl", "tr", "id", "ru", "sv", "hu", "ar", "hi", "vi", "zh", "ja"}

// English name → language → the names Natural Earth gives the places of that name.
type Gazetteer struct {
	names map[string]map[string]*cpcolls.Set[string]
}

var parenthesis = regexp.MustCompile(`\s*\([^)]*\)\s*`)

func cleaned(name string) string {
	return strings.Join(strings.Fields(parenthesis.ReplaceAllString(name, " ")), " ")
}

// Reads the layers the page's `npm run landmassNames` downloaded into the folder, and names the ones it lacks.
func Load(folder string) (*Gazetteer, []string, error) {
	gazetteer := &Gazetteer{names: map[string]map[string]*cpcolls.Set[string]{}}
	var missing []string
	for _, l := range layers {
		features, err := featuresOf(folder, l.name)
		if errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, l.name)
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		for _, properties := range features {
			gazetteer.add(properties, l.lower)
		}
	}
	return gazetteer, missing, nil
}

func (g *Gazetteer) add(properties map[string]any, lower bool) {
	field := func(name string) string {
		if !lower {
			name = strings.ToUpper(name)
		}
		value, _ := properties[name].(string)
		return cleaned(value)
	}
	for _, english := range []string{field("name"), field("name_en"), field("name_long")} {
		if english == "" {
			continue
		}
		for _, language := range languages {
			name := field("name_" + language)
			if name == "" {
				continue
			}
			if g.names[english] == nil {
				g.names[english] = map[string]*cpcolls.Set[string]{}
			}
			if g.names[english][language] == nil {
				g.names[english][language] = cpcolls.NewSet[string]()
			}
			g.names[english][language].Add(name)
		}
	}
}

// Each language's name for the place, where every place Natural Earth names so has the same one.
func (g *Gazetteer) NamesOf(english string) map[string]string {
	names := map[string]string{}
	for language, found := range g.names[cleaned(english)] {
		if found.Len() != 1 {
			continue
		}
		found.ForEach(func(name string) { names[language] = name })
	}
	return names
}

func featuresOf(folder, name string) ([]map[string]any, error) {
	path := filepath.Join(folder, name+".geojson")
	data, err := os.ReadFile(path) //nolint:gosec // the cache's own file.
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var collection struct {
		Features []struct {
			Properties map[string]any `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("%s does not read: %w", path, err)
	}
	features := make([]map[string]any, 0, len(collection.Features))
	for _, feature := range collection.Features {
		features = append(features, feature.Properties)
	}
	return features, nil
}
