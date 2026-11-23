package traps_query

import (
	"context"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

type Map interface {
	TrapsOf(flag string) ([]uint32, clicks.MapVersion)
}

type Flags interface {
	FlagOf(ctx context.Context, payer clicks.Payer) (string, error)
}

func NewMemoryQuery(traps Map, flags Flags) *MemoryQuery {
	return &MemoryQuery{traps: traps, flags: flags}
}

// MemoryQuery reads the traps from the map in memory, which keeps them with its tiles.
type MemoryQuery struct {
	traps Map
	flags Flags
}

func (q *MemoryQuery) TrapsOf(ctx context.Context, payer clicks.Payer) (*planetv1.GetTrapsResponse, error) {
	flag, err := q.flags.FlagOf(ctx, payer)
	if err != nil {
		return nil, fmt.Errorf("failed to read the flag the caller plays for: %w", err)
	}
	if flag == "" {
		return &planetv1.GetTrapsResponse{}, nil
	}

	tiles, version := q.traps.TrapsOf(flag)
	return &planetv1.GetTrapsResponse{CountryId: flag, TileIds: tiles, Version: planetmessage.Version(version)}, nil
}
