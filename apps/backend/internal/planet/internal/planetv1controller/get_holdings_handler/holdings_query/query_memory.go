package holdings_query

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

type Census interface {
	Census() (clicks.MapVersion, clicks.Census)
}

func NewMemoryQuery(census Census) *MemoryQuery {
	return &MemoryQuery{census: census}
}

type MemoryQuery struct {
	census Census
}

func (q *MemoryQuery) Holdings() *planetv1.GetHoldingsResponse {
	version, census := q.census.Census()

	return &planetv1.GetHoldingsResponse{
		Version:    planetmessage.Version(version),
		Countries:  planetmessage.Holdings(census.Countries),
		Landmasses: planetmessage.LandmassHoldings(census.Landmasses),
	}
}
