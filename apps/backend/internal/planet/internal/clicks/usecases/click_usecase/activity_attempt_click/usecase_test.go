package activity_attempt_click_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/activity_attempt_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const account = "00000000-0000-4000-8000-000000000001"

type answering struct {
	err error
}

func (a answering) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return click_usecase.Out{Limited: true}, a.err
}

// owners is the map before the click: tile to country, and every tile up to 100 exists.
type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) {
	if tile > 100 {
		return "", false
	}
	return o[tile], true
}

var now = time.Date(2026, 9, 30, 12, 0, 0, 123_456_000, time.UTC)

func try(t *testing.T, ctx context.Context, answer error, in click_usecase.In) (activity.Event, error) {
	t.Helper()

	recorded := &activity.Recorded{}
	out, err := activity_attempt_click.New(answering{err: answer}, recorded, owners{7: "bg"}, cptime.NewFixedClock(now)).
		Execute(ctx, in)

	assert.True(t, out.Limited, "the answer is the inner one, untouched")
	require.Len(t, recorded.Events(), 1)
	return recorded.Events()[0], err
}

func TestEveryTryIsRecordedWithItsAnswer(t *testing.T) {
	for _, tc := range []struct {
		name    string
		answer  error
		in      click_usecase.In
		outcome activity.Outcome
	}{
		{"accepted, or dropped by the ban with a zero Out", nil, click_usecase.In{TileID: 8, CountryID: "bg"}, activity.OutcomeAccepted},
		{"a tile its country held", nil, click_usecase.In{TileID: 7, CountryID: "bg"}, activity.OutcomeNoOp},
		{"a tile another country held", nil, click_usecase.In{TileID: 7, CountryID: "fr"}, activity.OutcomeAccepted},
		{"throttled", clicks.ErrThrottled, click_usecase.In{TileID: 8, CountryID: "bg"}, activity.OutcomeThrottled},
		{"an unknown country", fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, "zz"), click_usecase.In{TileID: 8, CountryID: "zz"}, activity.OutcomeInvalid},
		{"a tile off the map", clicks.ErrTileOutOfRange, click_usecase.In{TileID: 500, CountryID: "bg"}, activity.OutcomeInvalid},
		{"both bonuses", clicks.ErrBonusesTogether, click_usecase.In{TileID: 8, CountryID: "bg"}, activity.OutcomeInvalid},
		{"a fault of this server", errors.New("disk on fire"), click_usecase.In{TileID: 8, CountryID: "bg"}, activity.OutcomeFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event, err := try(t, cpctx.AddIPToContext(t.Context(), "2001:db8::9"), tc.answer, tc.in)

			assert.Equal(t, tc.answer, err, "the error is the inner one, untouched")
			assert.Equal(t, activity.Event{
				At: now, Kind: activity.KindClick, Caller: activity.Caller{Scope: "2001:db8::/64", Account: cpsession.NoAccount},
				Tile: tc.in.TileID, Country: tc.in.CountryID, Outcome: tc.outcome,
			}, event)
		})
	}
}

func TestATryCarriesTheAccountAndWhetherItSignedIn(t *testing.T) {
	ctx := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), account))

	event, err := try(t, ctx, clicks.ErrThrottled, click_usecase.In{TileID: 8, CountryID: "bg"})

	require.ErrorIs(t, err, clicks.ErrThrottled)
	assert.Equal(t, activity.Caller{
		Scope: "2001:db8::/64", Account: cpsession.AccountID(uuid.MustParse(account)), SignedIn: true,
	}, event.Caller)
}
