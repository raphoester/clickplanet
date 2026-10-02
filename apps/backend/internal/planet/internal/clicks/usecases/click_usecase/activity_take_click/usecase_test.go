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

type board map[uint32]string

func (b board) Owner(tile uint32) (string, bool) {
	if tile > 100 {
		return "", false
	}
	return b[tile], true
}

type rule struct {
	outcome clicks.Outcome
	err     error
}

func (r rule) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return click_usecase.Out{Limited: true, Outcome: r.outcome}, r.err
}

var now = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func click(t *testing.T, tiles board, inner rule, in click_usecase.In) []activity.Event {
	t.Helper()

	recorded := &activity.Recorded{}
	out, err := activity_take_click.New(inner, recorded, tiles, cptime.NewFixedClock(now)).
		Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), in)
	assert.Equal(t, inner.err, err, "the error is the inner one, untouched")
	assert.Equal(t, inner.outcome, out.Outcome, "the answer is the inner one, untouched")

	return recorded.Events()
}

func TestATakeIsRecordedWithWhoHeldTheTileBefore(t *testing.T) {
	events := click(t, board{7: "fr"}, rule{outcome: clicks.Taken}, click_usecase.In{TileID: 7, CountryID: "bg"})

	assert.Equal(t, []activity.Event{{
		At: now, Kind: activity.KindTake, Caller: activity.Caller{Scope: "2001:db8::/64", Account: cpsession.NoAccount},
		Tile: 7, Country: "bg", Held: "fr",
	}}, events)
}

func TestATakeOfATileNobodyHeldHoldsNobody(t *testing.T) {
	events := click(t, board{}, rule{outcome: clicks.Taken}, click_usecase.In{TileID: 7, CountryID: "bg"})

	require.Len(t, events, 1)
	assert.Empty(t, events[0].Held)
}

func TestAClearOnHomeSoilIsATakeThatSaysSo(t *testing.T) {
	events := click(t, board{7: "pl"}, rule{outcome: clicks.Cleared}, click_usecase.In{TileID: 7, CountryID: "de"})

	require.Len(t, events, 1)
	assert.True(t, events[0].Cleared)
	assert.Equal(t, "pl", events[0].Held)
	assert.Equal(t, "de", events[0].Country, "the flag clicked, though the tile now holds nobody")
}

func TestAClickThatChangedNothingIsNoTake(t *testing.T) {
	in := click_usecase.In{TileID: 7, CountryID: "bg"}

	assert.Empty(t, click(t, board{7: "bg"}, rule{outcome: clicks.Unchanged}, in), "a no-op")
	assert.Empty(t, click(t, board{}, rule{err: clicks.ErrUnknownCountry}, in), "a refusal")
	assert.Empty(t, click(t, board{}, rule{err: clicks.ErrTileOutOfRange}, click_usecase.In{TileID: 500, CountryID: "bg"}),
		"off the map")
}
