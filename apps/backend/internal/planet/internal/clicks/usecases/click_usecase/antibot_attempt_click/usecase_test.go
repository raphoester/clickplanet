package antibot_attempt_click_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/antibot_attempt_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fakeGuard struct {
	attempted []antibot.Click
}

func (g *fakeGuard) Attempted(click antibot.Click) { g.attempted = append(g.attempted, click) }

type throttled struct{}

func (throttled) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return click_usecase.Out{Limited: true}, clicks.ErrThrottled
}

func TestAntiBotAttemptClick(t *testing.T) {
	t.Run("shows the guard a click the throttle refuses", func(t *testing.T) {
		guard := &fakeGuard{}
		clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 18, 0, 0, 0, time.UTC))
		ctx := cpctx.AddIPToContext(t.Context(), "2001:db8::dead:beef")

		useCase := antibot_attempt_click.New(throttled{}, guard, clock)

		out, err := useCase.Execute(ctx, click_usecase.In{TileID: 42, CountryID: "BG"})

		require.ErrorIs(t, err, clicks.ErrThrottled, "the answer is the inner one, untouched")
		assert.True(t, out.Limited)
		require.Len(t, guard.attempted, 1)
		assert.Equal(t, antibot.Click{Scope: "2001:db8::/64", Tile: 42, Country: "BG", At: clock.Now()}, guard.attempted[0])
	})

	t.Run("shows the guard a linked account as signed in", func(t *testing.T) {
		guard := &fakeGuard{}
		clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 18, 0, 0, 0, time.UTC))
		ctx := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "a-player"))

		useCase := antibot_attempt_click.New(throttled{}, guard, clock)

		_, err := useCase.Execute(ctx, click_usecase.In{TileID: 42, CountryID: "BG"})

		require.ErrorIs(t, err, clicks.ErrThrottled)
		require.Len(t, guard.attempted, 1)
		assert.Equal(t, antibot.Click{Scope: "1.2.3.4", Account: "a-player", SignedIn: true, Tile: 42, Country: "BG", At: clock.Now()}, guard.attempted[0])
	})
}
