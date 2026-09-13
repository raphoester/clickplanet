package claim_bonus_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/claim_bonus_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type stubUseCase struct {
	out claim_bonus.Out
	err error

	in claim_bonus.In
}

func (s *stubUseCase) Execute(_ context.Context, in claim_bonus.In) (claim_bonus.Out, error) {
	s.in = in
	return s.out, s.err
}

func claim(t *testing.T, useCase claim_bonus_handler.UseCase) (*planetv1.ClaimBonusResponse, error) {
	t.Helper()

	res, err := claim_bonus_handler.New(useCase).ClaimBonus(t.Context(),
		connect.NewRequest(&planetv1.ClaimBonusRequest{Token: "a-token", CountryId: "fr"}))
	if err != nil {
		return nil, fmt.Errorf("claim refused: %w", err)
	}

	return res.Msg, nil
}

func TestAClaimAnswersTheWidenedAllowance(t *testing.T) {
	msg, err := claim(t, &stubUseCase{out: claim_bonus.Out{
		Budget:   toll.Budget{State: cpratelimit.State{Tokens: 7, Capacity: 30, PerSecond: 3}},
		Kind:     bonus.KindTripleClicks,
		Duration: time.Minute,
	}})
	require.NoError(t, err)

	assert.Equal(t, uint32(30), msg.GetBudget().GetCapacity())
	assert.InDelta(t, 3.0, msg.GetBudget().GetRefillPerSecond(), 1e-9)
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_TRIPLE_CLICKS, msg.GetKind())
	assert.Equal(t, uint32(60), msg.GetDurationSeconds())
}

func TestABombClaimSaysHowWideTheBlastIs(t *testing.T) {
	msg, err := claim(t, &stubUseCase{out: claim_bonus.Out{
		Kind: bonus.KindBomb, Duration: 30 * time.Second, BlastRadius: 0.03,
	}})
	require.NoError(t, err)

	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_BOMB, msg.GetKind())
	assert.InDelta(t, 0.03, msg.GetBlastRadius(), 1e-9)
}

func TestTheTokenAndCountryReachTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := claim(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, "a-token", useCase.in.Token)
	assert.Equal(t, "fr", useCase.in.CountryID)
}

func TestARefusedClaimIsNotFoundAndSaysNothingAboutWhy(t *testing.T) {
	_, err := claim(t, &stubUseCase{err: errors.New("lapsed, or never yours")})

	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.Contains(t, err.Error(), claim_bonus.ErrNoSuchBonus.Error())
	assert.NotContains(t, err.Error(), "lapsed")
}

func TestWithBoxesOffTheProcedureIsUnimplemented(t *testing.T) {
	_, err := claim(t, nil)

	require.Error(t, err)
	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestEveryKindTheServerGrantsHasAWireName(t *testing.T) {
	for _, kind := range bonus.Kinds {
		assert.NotEqualf(t, planetv1.BonusKind_BONUS_KIND_UNSPECIFIED, claim_bonus_handler.EncodeKind(kind),
			"%s would reach the client as a box it cannot draw", kind)
	}
}
