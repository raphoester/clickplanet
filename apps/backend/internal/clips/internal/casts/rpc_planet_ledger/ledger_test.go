package rpc_planet_ledger_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/rpc_planet_ledger"
)

var at = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a01"

type planet struct {
	pages map[string]*planetv1.GetLedgerResponse
	asked []*planetv1.GetLedgerRequest
}

func (p *planet) GetLedger(
	_ context.Context,
	req *connect.Request[planetv1.GetLedgerRequest],
) (*connect.Response[planetv1.GetLedgerResponse], error) {
	p.asked = append(p.asked, req.Msg)
	return connect.NewResponse(p.pages[req.Msg.GetPage()]), nil
}

func change(tile uint32, account string) *planetv1.LedgerChange {
	return &planetv1.LedgerChange{
		At: timestamppb.New(at), AccountId: account,
		Update: &planetv1.TileUpdate{TileId: tile, CountryId: "fr", PreviousCountryId: "de"},
	}
}

func TestEveryPageIsReadAndAChangeOfNoAccountIsLeftOut(t *testing.T) {
	ledger := &planet{pages: map[string]*planetv1.GetLedgerResponse{
		"":   {Changes: []*planetv1.LedgerChange{change(1, ada), change(2, "")}, NextPage: "77"},
		"77": {Changes: []*planetv1.LedgerChange{change(3, ada)}},
	}}
	window := casts.Window{Since: at.Add(-12 * time.Hour), Until: at}

	takes, err := rpc_planet_ledger.New(ledger).Takes(t.Context(), window)
	require.NoError(t, err)

	account, err := casts.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []casts.Take{
		{At: at, Tile: 1, Country: "fr", Previous: "de", Account: account},
		{At: at, Tile: 3, Country: "fr", Previous: "de", Account: account},
	}, takes)
	require.Len(t, ledger.asked, 2)
	assert.Equal(t, "77", ledger.asked[1].GetPage())
	assert.Equal(t, window.Since, ledger.asked[1].GetSince().AsTime())
}

func TestAChangeByWhatIsNotAnAccountFailsTheRead(t *testing.T) {
	ledger := &planet{pages: map[string]*planetv1.GetLedgerResponse{"": {Changes: []*planetv1.LedgerChange{change(1, "nobody")}}}}

	_, err := rpc_planet_ledger.New(ledger).Takes(t.Context(), casts.Window{Since: at.Add(-time.Hour), Until: at})

	assert.ErrorIs(t, err, casts.ErrInvalidAccount)
}
