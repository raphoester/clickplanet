package landmass_query

import (
	"context"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Who holds each tile is the map's, in memory: the ledger would read every change of every tile of the landmass.
type Holders interface {
	Holders(landmass clicks.LandmassID) []*planetv1.LandmassHolder
}

func NewMemoryQuery(holders Holders) *MemoryQuery {
	return &MemoryQuery{holders: holders}
}

type MemoryQuery struct {
	holders Holders
}

func (q *MemoryQuery) Landmass(_ context.Context, landmass clicks.LandmassID) (*planetv1.GetLandmassResponse, error) {
	return &planetv1.GetLandmassResponse{Holders: q.holders.Holders(landmass)}, nil
}
