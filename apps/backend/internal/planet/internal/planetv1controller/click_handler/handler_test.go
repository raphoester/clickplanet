package click_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type stubUseCase struct {
	out  click_usecase.Out
	err  error
	seen []click_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in click_usecase.In) (click_usecase.Out, error) {
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
	require.Equal(t, []click_usecase.In{{TileID: 42, CountryID: "fr"}}, useCase.seen)
}

func TestClickMapsTheBudget(t *testing.T) {
	state := clicks.Budget{State: cpratelimit.State{Tokens: 4.5, Capacity: 10, PerSecond: 2}}

	t.Run("carries the allowance when something throttles clicks", func(t *testing.T) {
		useCase := &stubUseCase{out: click_usecase.Out{Budget: state, Limited: true}}

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

	t.Run("shows a bank a trap left in debt below zero", func(t *testing.T) {
		useCase := &stubUseCase{out: click_usecase.Out{Budget: clicks.Budget{State: cpratelimit.State{Tokens: -7.5, Capacity: 60}}, Limited: true}}

		res, err := clickOn(t, useCase, &planetv1.ClickRequest{})

		require.NoError(t, err)
		assert.InDelta(t, -7.5, res.Msg.GetBudget().GetTokens(), 1e-9)
	})

	t.Run("a click refused for a debt carries the debt", func(t *testing.T) {
		state := clicks.Budget{State: cpratelimit.State{Tokens: -7.5, Capacity: 60, PerSecond: 0.2}}
		useCase := &stubUseCase{err: clicks.ErrThrottled, out: click_usecase.Out{Budget: state, Limited: true}}

		_, err := clickOn(t, useCase, &planetv1.ClickRequest{})

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		require.Len(t, connectErr.Details(), 1)
		value, valueErr := connectErr.Details()[0].Value()
		require.NoError(t, valueErr)
		budget, ok := value.(*planetv1.ClickBudget)
		require.True(t, ok)
		assert.InDelta(t, -7.5, budget.GetTokens(), 1e-9)
	})
}

func TestClickMapsTheErrors(t *testing.T) {
	for name, sentinel := range map[string]error{
		"an unknown country":          clicks.ErrUnknownCountry,
		"a tile out of range":         clicks.ErrTileOutOfRange,
		"spread and enclose together": clicks.ErrBonusesTogether,
	} {
		t.Run(name+" is the caller's fault", func(t *testing.T) {
			_, err := clickOn(t, &stubUseCase{err: sentinel}, &planetv1.ClickRequest{})

			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.ErrorIs(t, err, sentinel, "the use case's own sentence travels with it")
		})
	}

	t.Run("a throttled click is resource exhausted and carries the wait", func(t *testing.T) {
		state := clicks.Budget{State: cpratelimit.State{Tokens: 0.4, Capacity: 5, PerSecond: 2}}
		useCase := &stubUseCase{err: clicks.ErrThrottled, out: click_usecase.Out{Budget: state, Limited: true}}

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
}

func TestAClickRefusedForItsFlagSaysWhichFlagAndUntilWhen(t *testing.T) {
	until := time.Date(2026, 10, 17, 13, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{err: fmt.Errorf("wrapped: %w", allegiances.Refusal{Flag: "de", Until: until})}

	_, err := clickOn(t, useCase, &planetv1.ClickRequest{TileId: 1, CountryId: "fr"})

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Len(t, connectErr.Details(), 1)
	value, err := connectErr.Details()[0].Value()
	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.Allegiance{CountryId: "de", ChangeableAtUnixMs: until.UnixMilli()}, value))
}

func TestAClickOutOfItsFlagsReachInAnArenaSaysSo(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("%w: tile 7 for fr", clicks.ErrOutOfReach)}

	_, err := clickOn(t, useCase, &planetv1.ClickRequest{TileId: 7, CountryId: "fr"})

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Len(t, connectErr.Details(), 1)
	value, err := connectErr.Details()[0].Value()
	require.NoError(t, err)
	assert.True(t, proto.Equal(&planetv1.ReachRefusal{}, value))
}
