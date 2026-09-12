package geodesic_map

import (
	"encoding/binary"
	"fmt"
	"math"
)

// The blob format, mirroring apps/frontend/src/app/viewer/coordinatesBinary.ts. Little-endian:
// "CPCO" | uint32 version | uint32 tile count N | N*3 f32 positions | N*2 f32 uvs. This is the
// second implementation of one format, so it is a transcription rather than an improvement.
const (
	coordinatesMagic   = "CPCO"
	coordinatesVersion = 1
	coordinatesHeader  = 12

	// 3 position floats and 2 uv floats; nothing here wants the uvs, but the size check needs them.
	floatsPerTile = 5
)

// decodeCoordinates returns the tile positions in the blob's own order; the caller turns that
// order into tile ids.
func decodeCoordinates(blob []byte) ([]float32, error) {
	if len(blob) < coordinatesHeader {
		return nil, fmt.Errorf("coordinates blob is truncated: %d bytes", len(blob))
	}

	if magic := string(blob[:4]); magic != coordinatesMagic {
		return nil, fmt.Errorf("not a coordinates blob: expected magic %q, got %q", coordinatesMagic, magic)
	}

	if version := binary.LittleEndian.Uint32(blob[4:8]); version != coordinatesVersion {
		return nil, fmt.Errorf("unsupported coordinates format version %d, expected %d", version, coordinatesVersion)
	}

	size := binary.LittleEndian.Uint32(blob[8:12])
	if size == 0 {
		return nil, fmt.Errorf("coordinates blob declares zero tiles")
	}

	expected := uint64(coordinatesHeader) + uint64(size)*floatsPerTile*4
	if uint64(len(blob)) != expected {
		return nil, fmt.Errorf("coordinates blob size mismatch: %d tiles need %d bytes, got %d",
			size, expected, len(blob))
	}

	positions := make([]float32, size*3)
	for i := range positions {
		offset := coordinatesHeader + i*4
		positions[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[offset : offset+4]))
	}

	return positions, nil
}
