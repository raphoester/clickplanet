package grant_charge_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/grant_charge_handler"
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

func grant(t *testing.T, useCase *stubUseCase, kind planetv1.BonusKind, amount uint32) (uint32, error) {
	t.Helper()

	res, err := grant_charge_handler.New(useCase).GrantCharge(t.Context(), connect.NewRequest(&planetv1.GrantChargeRequest{
		AccountId: account, Kind: kind, Amount: amount,
	}))
	if err != nil {
		return 0, err //nolint:wrapcheck // the test reads the code.
	}

	return res.Msg.GetKept(), nil
}

func TestOneKindIsGrantedToTheAccount(t *testing.T) {
	for kind, held := range map[planetv1.BonusKind]bonuses.Held{
		planetv1.BonusKind_BONUS_KIND_SHIELDS:        {Shields: 10},
		planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS:  {SpreadClicks: 10},
		planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS: {Enclosures: 10},
		planetv1.BonusKind_BONUS_KIND_BOMB:           {Bomb: true},
		planetv1.BonusKind_BONUS_KIND_REFILL:         {Refill: true},
	} {
		useCase := &stubUseCase{}

		_, err := grant(t, useCase, kind, 10)
		require.NoError(t, err)

		assert.Equal(t, grant_charges_usecase.In{Account: account, Grant: held}, useCase.in, kind.String())
	}
}

func TestTheAnswerIsWhatFit(t *testing.T) {
	useCase := &stubUseCase{out: grant_charges_usecase.Out{Before: bonuses.Held{Shields: 25}, After: bonuses.Held{Shields: 30}}}

	kept, err := grant(t, useCase, planetv1.BonusKind_BONUS_KIND_SHIELDS, 10)

	require.NoError(t, err)
	assert.Equal(t, uint32(5), kept, "a stack of 30 takes 5 more")
}

func TestABombAlreadyHeldKeepsNothing(t *testing.T) {
	useCase := &stubUseCase{out: grant_charges_usecase.Out{Before: bonuses.Held{Bomb: true}, After: bonuses.Held{Bomb: true}}}

	kept, err := grant(t, useCase, planetv1.BonusKind_BONUS_KIND_BOMB, 1)

	require.NoError(t, err)
	assert.Zero(t, kept)
}

func TestAKindThatIsNoneIsRefused(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := grant(t, useCase, planetv1.BonusKind_BONUS_KIND_UNSPECIFIED, 1)

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorIs(t, err, grant_charge_handler.ErrUnknownKind)
	assert.Equal(t, grant_charges_usecase.In{}, useCase.in, "nothing reaches the use case")
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		bonuses.ErrInvalidHolder:  connect.CodeInvalidArgument,
		bonuses.ErrNothingToGrant: connect.CodeInvalidArgument,
		errors.New("boom"):        connect.CodeUnknown,
	} {
		_, got := grant(t, &stubUseCase{err: err}, planetv1.BonusKind_BONUS_KIND_SHIELDS, 1)

		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
