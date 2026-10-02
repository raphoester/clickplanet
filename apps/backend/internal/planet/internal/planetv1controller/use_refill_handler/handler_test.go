package use_refill_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/use_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/use_refill_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type stubUseCase struct {
	out use_refill_usecase.Out
	err error
	in  use_refill_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in use_refill_usecase.In) (use_refill_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func use(t *testing.T, useCase *stubUseCase) (*planetv1.UseRefillResponse, error) {
	t.Helper()

	res, err := use_refill_handler.New(useCase).UseRefill(t.Context(),
		connect.NewRequest(&planetv1.UseRefillRequest{CountryId: "fr"}))
	if err != nil {
		return nil, fmt.Errorf("refill refused: %w", err)
	}
	return res.Msg, nil
}

func TestARefillAnswersTheFullBankAndWhatIsLeft(t *testing.T) {
	useCase := &stubUseCase{out: use_refill_usecase.Out{
		Budget: clicks.Budget{State: cpratelimit.State{Tokens: 60, Capacity: 60, PerSecond: 0.2}},
		Held:   bonuses.Held{Bomb: true},
	}}

	msg, err := use(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, "fr", useCase.in.CountryID)
	assert.InDelta(t, 60, msg.GetBudget().GetTokens(), 1e-9)
	assert.False(t, msg.GetCharges().GetRefill())
	assert.True(t, msg.GetCharges().GetBomb())
}

func TestNoRefillIsNotFound(t *testing.T) {
	_, err := use(t, &stubUseCase{err: use_refill_usecase.ErrNoRefill})

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAFullBankIsAFailedPrecondition(t *testing.T) {
	_, err := use(t, &stubUseCase{err: use_refill_usecase.ErrBankFull})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
