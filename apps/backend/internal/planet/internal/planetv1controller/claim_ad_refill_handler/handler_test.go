package claim_ad_refill_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_ad_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/claim_ad_refill_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type stubUseCase struct {
	budget clicks.Budget
	err    error
	in     claim_ad_refill_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in claim_ad_refill_usecase.In) (clicks.Budget, error) {
	s.in = in
	return s.budget, s.err
}

func claim(t *testing.T, useCase *stubUseCase) (*planetv1.ClaimAdRefillResponse, error) {
	t.Helper()

	res, err := claim_ad_refill_handler.New(useCase).ClaimAdRefill(t.Context(),
		connect.NewRequest(&planetv1.ClaimAdRefillRequest{CountryId: "fr"}))
	if err != nil {
		return nil, fmt.Errorf("ad refill refused: %w", err)
	}
	return res.Msg, nil
}

func TestAnAdAnswersTheFullBank(t *testing.T) {
	useCase := &stubUseCase{budget: clicks.Budget{State: cpratelimit.State{Tokens: 60, Capacity: 60, PerSecond: 0.2}}}

	msg, err := claim(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, "fr", useCase.in.CountryID)
	assert.InDelta(t, 60, msg.GetBudget().GetTokens(), 1e-9)
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		claim_ad_refill_usecase.ErrAdRefillOff: connect.CodeFailedPrecondition,
		claim_ad_refill_usecase.ErrNoAccount:   connect.CodeNotFound,
		clicks.ErrBankFull:                     connect.CodeFailedPrecondition,
	} {
		_, refused := claim(t, &stubUseCase{err: err})

		assert.Equal(t, code, connect.CodeOf(refused), err.Error())
		assert.ErrorIs(t, refused, err)
	}
}

func TestASpanUsedSaysWhenTheNextMayBe(t *testing.T) {
	next := time.Date(2026, 10, 30, 13, 0, 0, 0, time.UTC)

	_, err := claim(t, &stubUseCase{err: bonuses.AdRefillRefusal{Until: next}})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	require.Len(t, refused.Details(), 1)
	detail, err := refused.Details()[0].Value()
	require.NoError(t, err)
	used, ok := detail.(*planetv1.AdRefillRefusal)
	require.True(t, ok)
	assert.Equal(t, next.UnixMilli(), used.GetAvailableAtUnixMs())
}
