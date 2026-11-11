package grant_charges_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/grant_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/grant_charges_handler"
)

const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

type stubUseCase struct {
	in  grant_charges_usecase.In
	out grant_charges_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in grant_charges_usecase.In) (grant_charges_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func grant(t *testing.T, useCase *stubUseCase, req *planetv1.GrantChargesRequest) (*planetv1.GrantChargesResponse, error) {
	t.Helper()

	res, err := grant_charges_handler.New(useCase).GrantCharges(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("grant refused: %w", err)
	}

	return res.Msg, nil
}

func TestEveryChargeAskedReachesTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := grant(t, useCase, &planetv1.GrantChargesRequest{
		AccountId: account, Refill: true, Bomb: true, Enclosures: 2, SpreadClicks: 5,
	})
	require.NoError(t, err)

	assert.Equal(t, grant_charges_usecase.In{
		Account: account,
		Grant:   charges.Held{Refill: true, Bomb: true, Enclosures: 2, SpreadClicks: 5},
	}, useCase.in)
}

func TestTheAnswerSaysWhatWasHeldBeforeAndAfter(t *testing.T) {
	useCase := &stubUseCase{out: grant_charges_usecase.Out{
		Holder: account,
		Before: charges.Held{SpreadClicks: 7},
		After:  charges.Held{Bomb: true, SpreadClicks: 8},
	}}

	res, err := grant(t, useCase, &planetv1.GrantChargesRequest{AccountId: account, Bomb: true, SpreadClicks: 4})
	require.NoError(t, err)

	assert.Equal(t, account, res.GetAccountId())
	assert.Equal(t, uint32(7), res.GetBefore().GetSpreadClicksLeft())
	assert.False(t, res.GetBefore().GetBomb())
	assert.Equal(t, uint32(8), res.GetAfter().GetSpreadClicksLeft())
	assert.True(t, res.GetAfter().GetBomb())
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("%w: %q", charges.ErrInvalidHolder, "bot"): connect.CodeInvalidArgument,
		charges.ErrNothingToGrant:                             connect.CodeInvalidArgument,
		charges.ErrNegativeGrant:                              connect.CodeInvalidArgument,
		errors.New("boom"):                                    connect.CodeUnknown,
	} {
		_, got := grant(t, &stubUseCase{err: err}, &planetv1.GrantChargesRequest{AccountId: "bot"})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
