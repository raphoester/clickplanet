// Package natural_earth reads the names Natural Earth gives a place in other languages, from the layers the page's
// landmass names were made of.
package natural_earth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// The landmass names were made from this release (the page's scripts/map/naturalEarth.mjs).
const Tag = "v5.1.2"

const base = "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/" + Tag + "/geojson/"

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
var languages = []string{"fr", "de", "es", "pt", "pl", "tr", "id"}

// English name → language → the names Natural Earth gives the places of that name.
type Gazetteer struct {
	names map[string]map[string]*cpcolls.Set[string]
}

var parenthesis = regexp.MustCompile(`\s*\([^)]*\)\s*`)

func cleaned(name string) string {
	return strings.Join(strings.Fields(parenthesis.ReplaceAllString(name, " ")), " ")
}

// Reads the layers from the folder, downloading the ones it lacks: the page's scripts keep the same cache.
func Load(ctx context.Context, folder string) (*Gazetteer, error) {
	gazetteer := &Gazetteer{names: map[string]map[string]*cpcolls.Set[string]{}}
	for _, l := range layers {
		features, err := featuresOf(ctx, folder, l.name)
		if err != nil {
			return nil, err
		}
		for _, properties := range features {
			gazetteer.add(properties, l.lower)
		}
	}
	return gazetteer, nil
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

var errNoLayer = errors.New("no such layer at Natural Earth")

func featuresOf(ctx context.Context, folder, name string) ([]map[string]any, error) {
	path := filepath.Join(folder, name+".geojson")
	data, err := os.ReadFile(path) //nolint:gosec // the cache's own file.
	if errors.Is(err, fs.ErrNotExist) {
		data, err = download(ctx, name, path)
	}
	if err != nil {
		return nil, err
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

func download(ctx context.Context, name, path string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+name+".geojson", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ask for %s: %w", name, err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", name, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s at %s: %s", errNoLayer, name, Tag, response.Status)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", name, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("failed to make the cache %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, fmt.Errorf("failed to keep %s: %w", path, err)
	}
	return data, nil
}
