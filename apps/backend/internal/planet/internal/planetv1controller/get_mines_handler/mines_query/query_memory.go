package mines_query

import (
	"context"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

type Map interface {
	MinesOf(flag string) ([]uint32, clicks.MapVersion)
}

type Flags interface {
	FlagOf(ctx context.Context, payer clicks.Payer) (string, error)
}

func NewMemoryQuery(mines Map, flags Flags) *MemoryQuery {
	return &MemoryQuery{mines: mines, flags: flags}
}

// MemoryQuery reads the mines from the map in memory, which keeps them with its tiles.
type MemoryQuery struct {
	mines Map
	flags Flags
}

func (q *MemoryQuery) MinesOf(ctx context.Context, payer clicks.Payer) (*planetv1.GetMinesResponse, error) {
	flag, err := q.flags.FlagOf(ctx, payer)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flag the caller plays for: %w", err)
	}
	if flag == "" {
		return &planetv1.GetMinesResponse{}, nil
	}

	tiles, version := q.mines.MinesOf(flag)
	return &planetv1.GetMinesResponse{CountryId: flag, TileIds: tiles, Version: planetmessage.Version(version)}, nil
}
