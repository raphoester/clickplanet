package antibot_challenge_click_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/antibot_challenge_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type asked struct {
	scope   string
	session string
}

type fakeGuard struct {
	challenged bool
	asked      []asked
}

func (g *fakeGuard) Challenged(scope, session string) bool {
	g.asked = append(g.asked, asked{scope: scope, session: session})
	return g.challenged
}

type fakeClick struct{ ran bool }

func (c *fakeClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	c.ran = true
	return click_usecase.Out{}, nil
}

func execute(t *testing.T, ctx context.Context, guard *fakeGuard, inner *fakeClick) error {
	t.Helper()

	useCase := antibot_challenge_click.New(inner, guard)
	_, err := useCase.Execute(ctx, click_usecase.In{TileID: 42, CountryID: "PS"})

	return err
}

func clicker(t *testing.T, ip, session string) context.Context {
	t.Helper()
	return cpctx.AddSessionIDToContext(cpctx.AddIPToContext(t.Context(), ip), session)
}

func TestAntiBotChallengeClick(t *testing.T) {
	t.Run("lets an unchallenged caller through", func(t *testing.T) {
		inner := &fakeClick{}

		require.NoError(t, execute(t, t.Context(), &fakeGuard{}, inner))
		assert.True(t, inner.ran)
	})

	t.Run("refuses a challenged caller by name, so a client can answer it", func(t *testing.T) {
		inner := &fakeClick{}

		err := execute(t, t.Context(), &fakeGuard{challenged: true}, inner)

		require.ErrorIs(t, err, clicks.ErrChallenged, "the opposite of a shadow ban: it is said out loud")
		assert.False(t, inner.ran, "and the map is not touched")
	})

	t.Run("asks about the scope the ban is keyed on and the session the edge accepted", func(t *testing.T) {
		guard := &fakeGuard{}

		require.NoError(t, execute(t, clicker(t, "2001:db8::dead:beef", "mint-1"), guard, &fakeClick{}))
		assert.Equal(t, []asked{{scope: "2001:db8::/64", session: "mint-1"}}, guard.asked)
	})

	t.Run("asks on every click, because asking is also what answers a challenge", func(t *testing.T) {
		guard := &fakeGuard{}

		for range 3 {
			require.NoError(t, execute(t, clicker(t, "1.2.3.4", "mint-2"), guard, &fakeClick{}))
		}

		assert.Len(t, guard.asked, 3)
	})
}
