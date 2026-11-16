package pay_level_up_handler_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/commands/pay_level_up_handler"
)

type useCase struct {
	paid []bonuses.LevelUp
}

func (u *useCase) Execute(_ context.Context, levelUp bonuses.LevelUp) error {
	u.paid = append(u.paid, levelUp)
	return nil
}

const (
	ada   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	grant = "6a1f3c2e-8d4b-4f7a-9e21-5b3c7d9e0f11"
)

func TestALevelUpIsReadInPlanetsWordsUnderProgressionsGrant(t *testing.T) {
	payer := &useCase{}

	err := pay_level_up_handler.New(payer).Handle(t.Context(), &bonusesv1.PayLevelUp{
		GrantId: grant, AccountId: ada, Level: 5,
		Unlocked: []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS, planetv1.BonusKind_BONUS_KIND_BOMB},
	})

	require.NoError(t, err)
	assert.Equal(t, []bonuses.LevelUp{{
		Account: ada, Grant: bonuses.GrantID(uuid.MustParse(grant)),
		Unlocked: bonuses.Unlocked{bonuses.KindEncloseClicks, bonuses.KindBomb},
	}}, payer.paid)
}

func TestACommandThatIsNotWholeIsRefused(t *testing.T) {
	for reason, command := range map[string]*bonusesv1.PayLevelUp{
		"no account": {GrantId: grant},
		"no grant":   {AccountId: ada},
		"a kind the bonuses never draw": {
			GrantId: grant, AccountId: ada, Unlocked: []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_UNSPECIFIED},
		},
	} {
		payer := &useCase{}

		require.Error(t, pay_level_up_handler.New(payer).Handle(t.Context(), command), reason)
		assert.Empty(t, payer.paid, reason)
	}
}
