package embedded_board

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Loader struct {
	expectTiles uint32
	logger      *slog.Logger
}

func New(expectTiles uint32, logger *slog.Logger) *Loader {
	return &Loader{expectTiles: expectTiles, logger: logger}
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
