// Package mapdata holds the tile coordinates and borders blobs, copied from the monorepo-shared /map by
// `make map` — this app's half of that source, the way generated/proto is its half of /proto.
//
// The directory is `map` to mirror the source it comes from; the package cannot be, since map is
// a keyword.
//
// It is embedded rather than read from disk: cmd/api is a self-contained container with no startup
// dependencies, and a file it has to find at boot is a boot that can fail for a reason nothing in
// this repo controls.
package mapdata

import (
	"embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
)

// Globbed because the file is content-addressed, which keeps `make map` a copy with no generated
// constant beside it, and makes drift between the two apps visible in the name.
//
//go:embed coordinates-*.bin borders-*.bin
var files embed.FS

// Coordinates returns the blob and the name it is content-addressed under.
func Coordinates() ([]byte, string, error) {
	return read("coordinates")
}

// Borders returns the tile to landmass table and the name it is content-addressed under.
func Borders() ([]byte, string, error) {
	return read("borders")
}

func read(asset string) ([]byte, string, error) {
	entries, err := fs.Glob(files, asset+"-*.bin")
	if err != nil {
		return nil, "", fmt.Errorf("failed to glob the %s blob: %w", asset, err)
	}
	if len(entries) != 1 {
		// Two means a `make map` that copied without sweeping the previous one.
		return nil, "", fmt.Errorf("expected exactly one %s blob, found %d: %v", asset, len(entries), entries)
	}

	blob, err := files.ReadFile(entries[0])
	if err != nil {
		return nil, "", fmt.Errorf("failed to read %s: %w", entries[0], err)
	}

	return blob, entries[0], nil
}

// DecodeBorders reads the borders blob: the region of each tile, tile ids from 1, and the country code of each
// region, "" for none. Planet builds its landmasses from them; the missions simulator, each tile's country.
func DecodeBorders(blob []byte) ([]uint16, []string, error) {
	if len(blob) < 4 {
		return nil, nil, errors.New("too short for a header length")
	}

	headerEnd := 4 + int(binary.LittleEndian.Uint32(blob))
	if headerEnd > len(blob) {
		return nil, nil, fmt.Errorf("header of %d bytes runs past the %d-byte blob", headerEnd-4, len(blob))
	}

	var header struct {
		Tiles int      `json:"tiles"`
		Codes []string `json:"codes"`
	}
	if err := json.Unmarshal(blob[4:headerEnd], &header); err != nil {
		return nil, nil, fmt.Errorf("failed to parse the header: %w", err)
	}

	if header.Tiles < 0 || headerEnd+header.Tiles*2 > len(blob) {
		return nil, nil, fmt.Errorf("%d tiles run past the %d-byte blob", header.Tiles, len(blob))
	}

	regions := make([]uint16, header.Tiles)
	for i := range regions {
		regions[i] = binary.LittleEndian.Uint16(blob[headerEnd+i*2:])
	}
	return regions, header.Codes, nil
}
