package click_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type stubUseCase struct {
	out  click.Out
	err  error
	seen []click.In
}

func (s *stubUseCase) Execute(_ context.Context, in click.In) (click.Out, error) {
	s.seen = append(s.seen, in)
	return s.out, s.err
}

func clickOn(
	t *testing.T,
	useCase *stubUseCase,
	req *planetv1.ClickRequest,
) (*connect.Response[planetv1.ClickResponse], error) {
	t.Helper()
	return click_handler.New(useCase).Click(t.Context(), connect.NewRequest(req))
}

func TestClickMapsTheRequest(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := clickOn(t, useCase, &planetv1.ClickRequest{TileId: 42, CountryId: "fr"})

	require.NoError(t, err)
	require.Equal(t, []click.In{{TileID: 42, CountryID: "fr"}}, useCase.seen)
}

func TestClickMapsTheBudget(t *testing.T) {
	state := cpratelimit.State{Tokens: 4.5, Capacity: 10, PerSecond: 2}

	t.Run("carries the allowance when something throttles clicks", func(t *testing.T) {
		useCase := &stubUseCase{out: click.Out{Budget: state, Limited: true}}

		res, err := clickOn(t, useCase, &planetv1.ClickRequest{})

		require.NoError(t, err)
		budget := res.Msg.GetBudget()
		require.NotNil(t, budget)
		assert.InDelta(t, 4.5, budget.GetTokens(), 1e-9)
		assert.Equal(t, uint32(10), budget.GetCapacity())
		assert.InDelta(t, float64(2), budget.GetRefillPerSecond(), 1e-9)
	})

	t.Run("promises no allowance when nothing throttles clicks", func(t *testing.T) {
		res, err := clickOn(t, &stubUseCase{}, &planetv1.ClickRequest{})

		require.NoError(t, err)
		assert.Nil(t, res.Msg.GetBudget())
	})

	t.Run("clamps a spent bucket at zero rather than showing it negative", func(t *testing.T) {
		useCase := &stubUseCase{out: click.Out{Budget: cpratelimit.State{Tokens: -3}, Limited: true}}

		res, err := clickOn(t, useCase, &planetv1.ClickRequest{})

		require.NoError(t, err)
		assert.InDelta(t, float64(0), res.Msg.GetBudget().GetTokens(), 1e-9)
	})
}

func TestClickMapsTheErrors(t *testing.T) {
	for name, sentinel := range map[string]error{
		"an unknown country":  clicks.ErrUnknownCountry,
		"a tile out of range": clicks.ErrTileOutOfRange,
	} {
		t.Run(name+" is the caller's fault", func(t *testing.T) {
			_, err := clickOn(t, &stubUseCase{err: sentinel}, &planetv1.ClickRequest{})

			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.ErrorIs(t, err, sentinel, "the use case's own sentence travels with it")
		})
	}

	t.Run("a throttled click is resource exhausted and carries the wait", func(t *testing.T) {
		state := cpratelimit.State{Tokens: 0.4, Capacity: 5, PerSecond: 2}
		useCase := &stubUseCase{err: clicks.ErrThrottled, out: click.Out{Budget: state, Limited: true}}

		_, err := clickOn(t, useCase, &planetv1.ClickRequest{})

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		require.Len(t, connectErr.Details(), 1, "a refusal has no response message to put the reading in")

		value, valueErr := connectErr.Details()[0].Value()
		require.NoError(t, valueErr)
		budget, ok := value.(*planetv1.ClickBudget)
		require.True(t, ok)
		assert.InDelta(t, 0.4, budget.GetTokens(), 1e-6)
	})

	t.Run("anything else is left for the error interceptor", func(t *testing.T) {
		cause := errors.New("disk on fire")

		_, err := clickOn(t, &stubUseCase{err: cause}, &planetv1.ClickRequest{})

		require.ErrorIs(t, err, cause, "the handler must not dress up what it does not recognise")
		require.Equal(t, connect.CodeUnknown, connect.CodeOf(err),
			"the handler picked no code: dressing this up would hide it from the interceptor's log")
	})
}
