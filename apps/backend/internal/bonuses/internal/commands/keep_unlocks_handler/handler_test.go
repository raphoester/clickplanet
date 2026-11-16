package keep_unlocks_handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/commands/keep_unlocks_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
)

type kept struct {
	account  bonuses.Account
	decision unlocks.Decision
}

type useCase struct {
	kept []kept
}

func (u *useCase) Execute(_ context.Context, account bonuses.Account, decision unlocks.Decision) error {
	u.kept = append(u.kept, kept{account: account, decision: decision})
	return decision.TimeError() //nolint:wrapcheck // the test reads the sentinel.
}

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var monday = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func TestTheKindsAreReadInPlanetsWords(t *testing.T) {
	keeper := &useCase{}

	err := keep_unlocks_handler.New(keeper).Handle(t.Context(), &bonusesv1.KeepUnlocks{
		AccountId: ada,
		Kinds:     []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS, planetv1.BonusKind_BONUS_KIND_BOMB},
		DecidedAt: timestamppb.New(monday),
	})

	require.NoError(t, err)
	require.Len(t, keeper.kept, 1)
	assert.Equal(t, bonuses.Account(ada), keeper.kept[0].account)
	assert.Equal(t, unlocks.DecisionOf(bonuses.Unlocked{bonuses.KindEncloseClicks, bonuses.KindBomb}, monday), keeper.kept[0].decision)
}

func TestACommandWithNoTimeIsRefused(t *testing.T) {
	err := keep_unlocks_handler.New(&useCase{}).Handle(t.Context(), &bonusesv1.KeepUnlocks{AccountId: ada})

	require.ErrorIs(t, err, unlocks.ErrNoTime)
}

func TestAKindTheBonusesModuleNeverDrawsIsRefused(t *testing.T) {
	keeper := &useCase{}

	err := keep_unlocks_handler.New(keeper).Handle(t.Context(), &bonusesv1.KeepUnlocks{
		AccountId: ada, Kinds: []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_UNSPECIFIED}, DecidedAt: timestamppb.New(monday),
	})

	require.ErrorIs(t, err, keep_unlocks_handler.ErrUnknownKind)
	assert.Empty(t, keeper.kept)
}

func TestABadAccountIsRefused(t *testing.T) {
	err := keep_unlocks_handler.New(&useCase{}).Handle(t.Context(), &bonusesv1.KeepUnlocks{AccountId: "ada", DecidedAt: timestamppb.New(monday)})

	require.ErrorIs(t, err, bonuses.ErrInvalidAccount)
}
