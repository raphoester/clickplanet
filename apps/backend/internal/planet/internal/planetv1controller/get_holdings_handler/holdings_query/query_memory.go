package holdings_query

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

type Tally interface {
	Tally() (clicks.MapVersion, clicks.Tally)
}

func NewMemoryQuery(tally Tally) *MemoryQuery {
	return &MemoryQuery{tally: tally}
}

type MemoryQuery struct {
	tally Tally
}

func (q *MemoryQuery) Holdings() *planetv1.GetHoldingsResponse {
	version, tally := q.tally.Tally()

	return &planetv1.GetHoldingsResponse{
		Version:    planetmessage.Version(version),
		Countries:  planetmessage.Holdings(tally.Countries),
		Landmasses: planetmessage.LandmassHoldings(tally.Landmasses),
	}
}
