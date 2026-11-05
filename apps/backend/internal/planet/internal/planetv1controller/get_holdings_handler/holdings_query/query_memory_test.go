package holdings_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_holdings_handler/holdings_query"
)

type stubTally struct {
	version clicks.MapVersion
	tally   clicks.Tally
}

func (s stubTally) Tally() (clicks.MapVersion, clicks.Tally) { return s.version, s.tally }

func TestTheHoldingsAreTheTallyAtItsVersion(t *testing.T) {
	answer := holdings_query.NewMemoryQuery(stubTally{
		version: clicks.MapVersion{Epoch: 1_790_000_000, Seq: 42},
		tally: clicks.Tally{
			Countries: []clicks.FlagTiles{{Flag: "fr", Tiles: 5}, {Flag: "de", Tiles: 1}},
			Landmasses: []clicks.LandmassTiles{
				{Landmass: 3, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 4}, {Flag: "de", Tiles: 1}}},
				{Landmass: 9, Flags: []clicks.FlagTiles{{Flag: "fr", Tiles: 1}}},
			},
		},
	}).Holdings()

	assert.True(t, proto.Equal(&planetv1.GetHoldingsResponse{
		Version:   &planetv1.MapVersion{Epoch: 1_790_000_000, Seq: 42},
		Countries: []*planetv1.Holding{{CountryId: "fr", Tiles: 5}, {CountryId: "de", Tiles: 1}},
		Landmasses: []*planetv1.LandmassHoldings{
			{LandmassId: 3, Holdings: []*planetv1.Holding{{CountryId: "fr", Tiles: 4}, {CountryId: "de", Tiles: 1}}},
			{LandmassId: 9, Holdings: []*planetv1.Holding{{CountryId: "fr", Tiles: 1}}},
		},
	}, answer), answer)
}
