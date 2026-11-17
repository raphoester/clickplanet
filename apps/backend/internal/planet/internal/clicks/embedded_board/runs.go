package embedded_board

import (
	"encoding/binary"
	"fmt"
)

const (
	runsMagic   = "CPRN"
	runsVersion = 1
	runsHeader  = 12
	runBytes    = 16
)

// Tiles side by side in one row of the lattice: the first and the last canonical id, both in.
type run struct{ first, last canonicalID }

func decodeRuns(blob []byte, points int) ([]run, error) {
	if len(blob) < runsHeader {
		return nil, fmt.Errorf("runs file is truncated: %d bytes", len(blob))
	}
	if magic := string(blob[:4]); magic != runsMagic {
		return nil, fmt.Errorf("not a runs file: expected magic %q, got %q", runsMagic, magic)
	}
	if version := binary.LittleEndian.Uint32(blob[4:8]); version != runsVersion {
		return nil, fmt.Errorf("unsupported runs format version %d, expected %d", version, runsVersion)
	}

	count := uint64(binary.LittleEndian.Uint32(blob[8:12]))
	if expected := runsHeader + count*runBytes; uint64(len(blob)) != expected {
		return nil, fmt.Errorf("runs file size mismatch: %d runs need %d bytes, got %d", count, expected, len(blob))
	}

	runs := make([]run, count)
	after := uint64(0)
	for k := range runs {
		at := runsHeader + k*runBytes
		first := binary.LittleEndian.Uint64(blob[at:])
		last := binary.LittleEndian.Uint64(blob[at+8:])
		if last < first || last >= uint64(points) {
			return nil, fmt.Errorf("run %d-%d is not a range of the lattice's %d points", first, last, points)
		}
		if k > 0 && first <= after {
			return nil, fmt.Errorf("run %d-%d is not after the run before it", first, last)
		}
		after = last
		runs[k] = run{first: canonicalID(first), last: canonicalID(last)} //nolint:gosec // below the lattice's size.
	}

	return runs, nil
}
