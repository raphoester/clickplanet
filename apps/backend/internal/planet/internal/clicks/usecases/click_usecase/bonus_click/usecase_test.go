package bonus_click_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/bonus_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type stubClick struct {
	out click_usecase.Out
	err error
}

func (s stubClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return s.out, s.err
}

type recordingPresence struct {
	clicks []bonuses.Click
}

func (r *recordingPresence) Clicked(click bonuses.Click) {
	r.clicks = append(r.clicks, click)
}

func leaving(tokens float64) click_usecase.Out {
	return click_usecase.Out{Budget: clicks.Budget{State: cpratelimit.State{Tokens: tokens, Capacity: 60}}, Limited: true}
}

func TestAnAcceptedClickMarksTheCallerAsPlaying(t *testing.T) {
	presence := &recordingPresence{}

	_, err := bonus_click.New(stubClick{}, presence).Execute(t.Context(), click_usecase.In{})
	require.NoError(t, err)

	assert.Len(t, presence.clicks, 1)
}

func TestARefusedClickIsNotPlaying(t *testing.T) {
	presence := &recordingPresence{}

	_, err := bonus_click.New(stubClick{err: errors.New("nope")}, presence).Execute(t.Context(), click_usecase.In{})

	require.Error(t, err)
	assert.Empty(t, presence.clicks, "a click that changed nothing is not playing")
}

func TestAClickSaysWhichAccountPlaysBehindTheScope(t *testing.T) {
	presence := &recordingPresence{}
	born := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	ctx := cpctx.AddAccountCreatedToContext(
		cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-guest"), born)

	_, err := bonus_click.New(stubClick{}, presence).Execute(ctx, click_usecase.In{})
	require.NoError(t, err)

	assert.Equal(t, []bonuses.Click{{Caller: bonuses.Caller{Scope: "1.2.3.4", Account: "a-guest"}, Born: born}},
		presence.clicks)
}

func TestALinkedAccountPlaysOnItsOwnSchedule(t *testing.T) {
	presence := &recordingPresence{}
	ctx := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-player"))

	_, err := bonus_click.New(stubClick{}, presence).Execute(ctx, click_usecase.In{})
	require.NoError(t, err)

	require.Len(t, presence.clicks, 1)
	assert.Equal(t, bonuses.Caller{Scope: "1.2.3.4", Account: "a-player", Linked: true}, presence.clicks[0].Caller)
	assert.Equal(t, bonuses.Entrant("account:a-player"), bonuses.EntrantOf(presence.clicks[0].Caller))
}

func TestAClickThatLeavesLessThanOneClickInTheBankIsDry(t *testing.T) {
	for _, tc := range []struct {
		name string
		out  click_usecase.Out
		dry  bool
	}{
		{name: "nothing left", out: leaving(0), dry: true},
		{name: "part of a click left", out: leaving(0.9), dry: true},
		{name: "one click left", out: leaving(1)},
		{name: "no throttle, so no reading", out: click_usecase.Out{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			presence := &recordingPresence{}

			_, err := bonus_click.New(stubClick{out: tc.out}, presence).Execute(t.Context(), click_usecase.In{})
			require.NoError(t, err)

			require.Len(t, presence.clicks, 1)
			assert.Equal(t, tc.dry, presence.clicks[0].Dry)
		})
	}
}

// The offers hear exactly ClickTried's TAKEN, STRUCK and UNCHANGED: a try the throttle or the chain inside it refused or
// failed is not heard, and the gates outside it (the refusal, the flag, the reach) never reach it.
func TestOnlyATakeAStrikeOrAnUnchangedTileIsHeard(t *testing.T) {
	for _, outcome := range []clicks.Outcome{clicks.Taken, clicks.Shielded, clicks.Unchanged} {
		presence := &recordingPresence{}

		_, err := bonus_click.New(stubClick{out: click_usecase.Out{Outcome: outcome}}, presence).
			Execute(t.Context(), click_usecase.In{})
		require.NoError(t, err)

		assert.Len(t, presence.clicks, 1, outcome)
	}

	for _, refused := range []error{
		clicks.ErrThrottled, clicks.ErrUnknownCountry, clicks.ErrTileOutOfRange, clicks.ErrBonusesTogether,
		errors.New("the spread failed"),
	} {
		presence := &recordingPresence{}

		_, err := bonus_click.New(stubClick{err: refused}, presence).Execute(t.Context(), click_usecase.In{})
		require.ErrorIs(t, err, refused)

		assert.Empty(t, presence.clicks, refused)
	}
}
