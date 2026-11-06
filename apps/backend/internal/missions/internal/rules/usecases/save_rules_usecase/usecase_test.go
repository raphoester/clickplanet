package save_rules_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules/usecases/save_rules_usecase"
)

type recordingStore struct {
	saved []rules.Rules
}

func (s *recordingStore) Save(_ context.Context, kept rules.Rules) error {
	s.saved = append(s.saved, kept)
	return nil
}

func TestNumbersThatPlayAreKept(t *testing.T) {
	store := &recordingStore{}
	in := &missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{GoalPercent: 40, LeastGoal: 300, FewestPoints: 2, MostPoints: 8}}

	require.NoError(t, save_rules_usecase.New(store).Execute(t.Context(), in))

	want, err := rules.RulesOf(in)
	require.NoError(t, err)
	assert.Equal(t, []rules.Rules{want}, store.saved)
}

func TestNumbersThatCannotPlayAreRefusedAndNothingIsKept(t *testing.T) {
	store := &recordingStore{}

	err := save_rules_usecase.New(store).Execute(t.Context(),
		&missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{GoalPercent: 40, LeastGoal: 300, FewestPoints: 9, MostPoints: 8}})

	require.ErrorIs(t, err, rules.ErrInvalidRules)
	assert.Empty(t, store.saved)
}
