package embedded_board

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
)

type Loader struct {
	expectTiles uint32
	logger      *slog.Logger
}

func New(expectTiles uint32, logger *slog.Logger) *Loader {
	return &Loader{expectTiles: expectTiles, logger: logger}
}

// Carried is the board this process plays, as its files name it: its directory, and the hash of its positions in tile
// order (the manifest's positionsHash, which LoadGeography checks the rebuilt positions against). Planet's row of it
// names the same hash, or the boot refuses it.
func (l *Loader) Carried() (eras.Carried, error) {
	board, err := l.board()
	if err != nil {
		return eras.Carried{}, err
	}
	return eras.Carried{Name: clicks.BoardName(board.name), Hash: fmt.Sprintf("%08x", board.manifest.PositionsHash)}, nil
}

func (l *Loader) LoadGeography() (*clicks.Geography, error) {
	started := time.Now()

	board, err := l.board()
	if err != nil {
		return nil, err
	}

	positions, tileOf, err := board.tiles()
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild board %s's tiles: %w", board.name, err)
	}

	geography, err := clicks.NewGeography(positions, latticeEdges(board.manifest.Lattice.Cols, tileOf))
	if err != nil {
		return nil, fmt.Errorf("failed to build the geography of board %s: %w", board.name, err)
	}

	stats := geography.Stats()
	l.logger.Info("map geography loaded",
		slog.String("board", board.name),
		slog.String("asset", board.asset),
		slog.Int("tiles", int(stats.Tiles)),
		slog.Int("edges", int(stats.Edges)),
		slog.Any("degrees", stats.Degrees),
		slog.Any("took", time.Since(started).Round(time.Millisecond)),
	)

	return geography, nil
}
