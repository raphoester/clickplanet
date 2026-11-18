package embedded_board

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpboards"
)

const (
	provincesMagic   = "CPPV"
	provincesVersion = 1
	provincesHeader  = 20
	// Five float32s a province: its frame, which the page alone reads.
	frameBytes = 20
)

// loadProvinces refuses a provinces file cut on other borders than the board's. A board with none has none.
func loadProvinces(board cpboards.Board, borders *clicks.Borders) (*clicks.Provinces, error) {
	asset := board.Manifest.Files.Provinces
	if asset == "" {
		return clicks.NoProvinces(), nil
	}
	blob, err := board.File(asset)
	if err != nil {
		return nil, err //nolint:wrapcheck // the board names the file.
	}
	cutOn, err := bordersHashOf(board.Manifest.Files.Borders)
	if err != nil {
		return nil, err
	}

	provinces, err := decodeProvinces(asset, blob, cutOn, borders)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", asset, err)
	}
	return provinces, nil
}

// "borders-0ae1a037.bin" is 0x0ae1a037: the provinces file names the borders it was cut on so.
func bordersHashOf(asset string) (uint32, error) {
	hash, found := strings.CutPrefix(strings.TrimSuffix(asset, ".bin"), "borders-")
	if !found || len(hash) != 8 {
		return 0, fmt.Errorf("%q is not a borders file's name", asset)
	}
	parsed, err := strconv.ParseUint(hash, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is not a borders file's name: %w", asset, err)
	}
	return uint32(parsed), nil
}

// "CPPV" | uint32 version | uint32 tiles | uint32 provinces, none included | uint32 the borders' hash |
// provinces × 5 float32 frame | provinces × uint32 tiles | provinces × uint16 landmass | tiles × uint16 province.
func decodeProvinces(asset string, blob []byte, cutOn uint32, borders *clicks.Borders) (*clicks.Provinces, error) {
	if len(blob) < provincesHeader {
		return nil, fmt.Errorf("truncated: %d bytes", len(blob))
	}
	if magic := string(blob[:4]); magic != provincesMagic {
		return nil, fmt.Errorf("expected magic %q, got %q", provincesMagic, magic)
	}
	if version := binary.LittleEndian.Uint32(blob[4:]); version != provincesVersion {
		return nil, fmt.Errorf("unsupported version %d, expected %d", version, provincesVersion)
	}
	tiles := int(binary.LittleEndian.Uint32(blob[8:]))
	count := int(binary.LittleEndian.Uint32(blob[12:]))
	if on := binary.LittleEndian.Uint32(blob[16:]); on != cutOn {
		return nil, fmt.Errorf("cut on borders %08x, the board's are %08x", on, cutOn)
	}
	if want := provincesHeader + count*(frameBytes+4+2) + tiles*2; len(blob) != want {
		return nil, fmt.Errorf("%d provinces of %d tiles need %d bytes, got %d", count, tiles, want, len(blob))
	}

	at := provincesHeader + count*frameBytes
	sizes := make([]uint32, count)
	for k := range sizes {
		sizes[k] = binary.LittleEndian.Uint32(blob[at+k*4:])
	}
	at += count * 4
	landmasses := make([]uint16, count)
	for k := range landmasses {
		landmasses[k] = binary.LittleEndian.Uint16(blob[at+k*2:])
	}
	at += count * 2
	ofTile := make([]uint16, tiles)
	for i := range ofTile {
		ofTile[i] = binary.LittleEndian.Uint16(blob[at+i*2:])
	}

	provinces, err := clicks.NewProvinces(asset, borders, ofTile, landmasses, sizes)
	if err != nil {
		return nil, fmt.Errorf("failed to build the provinces: %w", err)
	}
	return provinces, nil
}
