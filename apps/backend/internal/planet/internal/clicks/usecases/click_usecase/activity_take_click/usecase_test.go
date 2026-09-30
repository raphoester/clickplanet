package activity_take_click_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/activity_take_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// board is the map: a click the rule accepts is written to it, as the real one does.
type board map[uint32]string

func (b board) Owner(tile uint32) (string, bool) {
	if tile > 100 {
		return "", false
	}
	return b[tile], true
}

type rule struct {
	board board
	err   error
}

func (r rule) Execute(_ context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if r.err != nil {
		return click_usecase.Out{}, r.err
	}
	r.board[in.TileID] = in.CountryID
	return click_usecase.Out{Limited: true}, nil
}

var now = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func click(t *testing.T, tiles board, err error, in click_usecase.In) []activity.Event {
	t.Helper()

	recorded := &activity.Recorded{}
	_, got := activity_take_click.New(rule{board: tiles, err: err}, recorded, tiles, cptime.NewFixedClock(now)).
		Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), in)
	require.ErrorIs(t, got, err)

	return recorded.Events()
}

func TestATakeIsRecordedWithWhoHeldTheTileBefore(t *testing.T) {
	events := click(t, board{7: "fr"}, nil, click_usecase.In{TileID: 7, CountryID: "bg"})

	assert.Equal(t, []activity.Event{{
		At: now, Kind: activity.KindTake, Caller: activity.Caller{Scope: "2001:db8::/64", Account: cpsession.NoAccount},
		Tile: 7, Country: "bg", Held: "fr",
	}}, events)
}

func TestATakeOfATileNobodyHeldHoldsNobody(t *testing.T) {
	events := click(t, board{}, nil, click_usecase.In{TileID: 7, CountryID: "bg"})

	require.Len(t, events, 1)
	assert.Empty(t, events[0].Held)
}

func TestAClickThatChangedNothingIsNoTake(t *testing.T) {
	assert.Empty(t, click(t, board{7: "bg"}, nil, click_usecase.In{TileID: 7, CountryID: "bg"}), "a no-op")
	assert.Empty(t, click(t, board{}, clicks.ErrUnknownCountry, click_usecase.In{TileID: 7, CountryID: "zz"}), "a refusal")
	assert.Empty(t, click(t, board{}, clicks.ErrTileOutOfRange, click_usecase.In{TileID: 500, CountryID: "bg"}), "off the map")
}
