package mines_query_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_mines_handler/mines_query"
)

type minefield map[string][]uint32

func (m minefield) MinesOf(flag string) ([]uint32, clicks.MapVersion) {
	return m[flag], clicks.MapVersion{Epoch: 7, Seq: 12}
}

type playsFor struct {
	flag string
}

func (p playsFor) FlagOf(context.Context, clicks.Payer) (string, error) { return p.flag, nil }

var mines = minefield{"fr": {5, 9}, "de": {30}}

func TestTheCallerReadsTheMinesOfTheFlagItPlaysForAndNoOther(t *testing.T) {
	answer, err := mines_query.NewMemoryQuery(mines, playsFor{flag: "fr"}).MinesOf(t.Context(), clicks.Payer{})

	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.GetMinesResponse{
		CountryId: "fr", TileIds: []uint32{5, 9}, Version: &planetv1.MapVersion{Epoch: 7, Seq: 12},
	}, answer), "%v", answer)
}

func TestACallerOfNoFlagReadsNoMine(t *testing.T) {
	answer, err := mines_query.NewMemoryQuery(mines, playsFor{}).MinesOf(t.Context(), clicks.Payer{})

	require.NoError(t, err)
	assert.Empty(t, answer.GetTileIds())
}
