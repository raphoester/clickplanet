// Package cpboards reads the boards this app carries (generated/map, a copy of /map/boards) and says how each compares
// with earth-1: a count of tiles scales with a board's density, a length with its spacing (docs/boards.md, "Rules at the
// new density"). Planet builds its geography from them; the other modules scale what they size from the board an
// answer or an event names.
package cpboards

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
)

var ErrUnknownBoard = errors.New("this app carries no such board")

// Manifest is a board's board-<hash>.json (/map/README.md#format).
type Manifest struct {
	Name    string `json:"name"`
	Body    string `json:"body"`
	Lattice struct {
		Cols int `json:"cols"`
		// The warp of each point's barycentric weights: null on earth-1.
		Warp *Warp `json:"warp"`
		// Three's 12 corners turned by the board's rotation, x, y, z each: null on earth-1.
		Corners []float64 `json:"corners"`
		// "three": tile ids are ranks in three's walk of the lattice; "canonical": in canonical id order.
		Order string `json:"order"`
	} `json:"lattice"`
	Tiles uint32 `json:"tiles"`
	// The median arc between two touching tiles, in radians.
	Spacing       float64 `json:"spacing"`
	PositionsHash uint32  `json:"positionsHash"`
	Files         struct {
		Runs     string `json:"runs"`
		Borders  string `json:"borders"`
		Capitals string `json:"capitals"`
		Bridges  string `json:"bridges"`
		Terrain  string `json:"terrain"`
	} `json:"files"`
}

// Warp is a board's warp of a point's barycentric weights w: h_k = w_k·(1 + a·w_k + b·w_k²) + c·wa·wb·wc, then each h_k
// over their sum (docs/boards.md, "The geometry").
type Warp struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
	C float64 `json:"c"`
}

// Board is a board this app carries: its manifest, and the file the manifest is.
type Board struct {
	Manifest Manifest
	// The manifest's own file, content-addressed: a stale copy shows in its name.
	Asset string
	files fs.FS
}

// Files are the boards this app carries, a directory each: generated/map's, and the tests' fixture under the testing
// tag.
func Files() fs.FS {
	return withFixture(mapdata.FS())
}

// Boards reads every board in files, by name.
func Boards(files fs.FS) ([]Board, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to list the boards: %w", err)
	}
	boards := make([]Board, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		board, err := read(files, entry.Name())
		if err != nil {
			return nil, err
		}
		boards = append(boards, board)
	}
	slices.SortFunc(boards, func(a, b Board) int { return strings.Compare(a.Manifest.Name, b.Manifest.Name) })
	return boards, nil
}

// Named is the board of that name in files: ErrUnknownBoard when there is none.
func Named(files fs.FS, name string) (Board, error) {
	if name == "" || path.Base(name) != name {
		return Board{}, fmt.Errorf("%w: %q", ErrUnknownBoard, name)
	}
	if _, err := fs.Stat(files, name); err != nil {
		return Board{}, fmt.Errorf("%w: %q", ErrUnknownBoard, name)
	}
	return read(files, name)
}

func read(files fs.FS, name string) (Board, error) {
	// Globbed: the name is content-addressed, so a stale copy shows in it.
	manifests, err := fs.Glob(files, path.Join(name, "board-*.json"))
	if err != nil {
		return Board{}, fmt.Errorf("failed to glob board %s's manifest: %w", name, err)
	}
	if len(manifests) != 1 {
		// Two means a `make map` that copied without sweeping the previous one.
		return Board{}, fmt.Errorf("expected exactly one manifest for board %s, found %d: %v", name, len(manifests), manifests)
	}
	blob, err := fs.ReadFile(files, manifests[0])
	if err != nil {
		return Board{}, fmt.Errorf("failed to read %s: %w", manifests[0], err)
	}
	var manifest Manifest
	if err := json.Unmarshal(blob, &manifest); err != nil {
		return Board{}, fmt.Errorf("failed to parse %s: %w", manifests[0], err)
	}
	asset := path.Base(manifests[0])
	if manifest.Name != name {
		return Board{}, fmt.Errorf("%s names board %q, in the directory of %q", asset, manifest.Name, name)
	}
	if manifest.Spacing <= 0 {
		return Board{}, fmt.Errorf("%s has no spacing", asset)
	}
	return Board{Manifest: manifest, Asset: asset, files: files}, nil
}

// File is one of the files the board's manifest names.
func (b Board) File(name string) ([]byte, error) {
	if path.Base(name) != name {
		return nil, fmt.Errorf("%q is not a file of board %s", name, b.Manifest.Name)
	}
	blob, err := fs.ReadFile(b.files, path.Join(b.Manifest.Name, name))
	if err != nil {
		return nil, fmt.Errorf("failed to read board %s's %s: %w", b.Manifest.Name, name, err)
	}
	return blob, nil
}

// Hash is the board's identity, planet's row of it names it: its manifest's positionsHash in 8 hex. It moves exactly
// when a tile's id or place does.
func (b Board) Hash() string {
	return fmt.Sprintf("%08x", b.Manifest.PositionsHash)
}

// Scale is how the board compares with earth-1.
func (b Board) Scale() Scale {
	return Scale{length: b.Manifest.Spacing / Earth1Spacing}
}
