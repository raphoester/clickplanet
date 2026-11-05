package apply_grant_handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/apply_grant_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/apply_grant_handler"
)

type recordingUseCase struct {
	applied []apply_grant_usecase.In
}

func (u *recordingUseCase) Execute(_ context.Context, in apply_grant_usecase.In) error {
	u.applied = append(u.applied, in)
	return nil
}

const grant = "6f1c2a3e-0d4b-4e5f-8a9b-1c2d3e4f5a6b"

func TestEachKindIsGrantedAsItsCharge(t *testing.T) {
	for wire, kind := range map[planetv1.BonusKind]bonuses.Kind{
		planetv1.BonusKind_BONUS_KIND_SHIELDS:        bonuses.KindShields,
		planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS:  bonuses.KindSpreadClicks,
		planetv1.BonusKind_BONUS_KIND_BOMB:           bonuses.KindBomb,
		planetv1.BonusKind_BONUS_KIND_REFILL:         bonuses.KindRefill,
		planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS: bonuses.KindEncloseClicks,
	} {
		useCase := &recordingUseCase{}

		require.NoError(t, apply_grant_handler.New(useCase).Handle(t.Context(),
			&planetv1.GrantCharges{GrantId: grant, AccountId: "the-account", Kind: wire, Amount: 4}))

		id, err := bonuses.GrantIDOf(grant)
		require.NoError(t, err)
		assert.Equal(t, []apply_grant_usecase.In{{Grant: id, Account: "the-account", Kind: kind, Amount: 4}}, useCase.applied)
	}
}

func TestACommandThatNamesNoGrantOrNoKindIsRefused(t *testing.T) {
	useCase := &recordingUseCase{}

	assert.ErrorIs(t, apply_grant_handler.New(useCase).Handle(t.Context(),
		&planetv1.GrantCharges{GrantId: "nope", Kind: planetv1.BonusKind_BONUS_KIND_SHIELDS}), bonuses.ErrInvalidGrant)
	assert.ErrorIs(t, apply_grant_handler.New(useCase).Handle(t.Context(),
		&planetv1.GrantCharges{GrantId: grant}), apply_grant_handler.ErrUnknownKind)
	assert.Empty(t, useCase.applied)
}
