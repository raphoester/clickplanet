// Command flagpins writes generated/flags/pins.json: the capital of each flag the game ships with that is a region of a
// country (England, Scotland, Wales), read from Natural Earth as the map's capitals are. `make flag-pins` runs it.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/cmd/flagpins/internal/regions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

// The map's tag (apps/frontend/scripts/map/naturalEarth.mjs): the countries' capitals come from the same places.
const naturalEarth = "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/v5.1.2/geojson/"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	path, err := parseFlags(args)
	if err != nil {
		return err
	}

	pins, err := pick(context.Background())
	if err != nil {
		return err
	}

	return write(path, pins, out)
}

func parseFlags(args []string) (string, error) {
	set := flag.NewFlagSet("flagpins", flag.ContinueOnError)
	path := set.String("out", "generated/flags/pins.json", "the file to write")
	if err := set.Parse(args); err != nil {
		return "", fmt.Errorf("usage: flagpins [-out file]: %w", err)
	}
	return *path, nil
}

func pick(ctx context.Context) (map[string]regions.Pin, error) {
	subunits, err := fetch(ctx, "ne_10m_admin_0_map_subunits")
	if err != nil {
		return nil, err
	}
	places, err := fetch(ctx, "ne_10m_populated_places")
	if err != nil {
		return nil, err
	}

	var codes []string
	for _, built := range cpflags.BuiltIn() {
		codes = append(codes, built.Code)
	}
	return regions.Pins(codes, subunits, places) //nolint:wrapcheck // the regions name the flag.
}

func fetch(ctx context.Context, layer string) (regions.Layer, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, naturalEarth+layer+".geojson", nil)
	if err != nil {
		return regions.Layer{}, fmt.Errorf("failed to ask for %s: %w", layer, err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return regions.Layer{}, fmt.Errorf("failed to fetch %s: %w", layer, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return regions.Layer{}, fmt.Errorf("failed to fetch %s: %s", layer, response.Status)
	}

	var decoded regions.Layer
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return regions.Layer{}, fmt.Errorf("failed to read %s: %w", layer, err)
	}
	return decoded, nil
}

func write(path string, pins map[string]regions.Pin, out io.Writer) error {
	file, err := json.MarshalIndent(pins, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to write the pins: %w", err)
	}
	if err := os.WriteFile(path, append(file, '\n'), 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	for code, pin := range pins {
		if _, err := fmt.Fprintf(out, "%s: %s (%v, %v)\n", code, pin.Place, pin.Latitude, pin.Longitude); err != nil {
			return fmt.Errorf("failed to show the pins: %w", err)
		}
	}
	return nil
}
