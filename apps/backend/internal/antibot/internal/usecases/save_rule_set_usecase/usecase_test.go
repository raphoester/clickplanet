package save_rule_set_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/save_rule_set_usecase"
)

var rush = rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 50 })

func TestASetIsKeptAsWritten(t *testing.T) {
	sets := rules.NewMemoryStore(rules.Seeded())

	require.NoError(t, save_rule_set_usecase.New(sets, guard.Parts()).Execute(t.Context(), rush))

	kept, ok := sets.Kept("rush")
	require.True(t, ok)
	assert.True(t, proto.Equal(rush.Rules, kept.Rules))
}

func TestASetThatCannotPlayIsNotKept(t *testing.T) {
	sets := rules.NewMemoryStore(rules.Seeded())

	err := save_rule_set_usecase.New(sets, guard.Parts()).Execute(t.Context(),
		rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 1 }))

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	_, ok := sets.Kept("rush")
	assert.False(t, ok)
}

func TestASaveThatFailsSaysSo(t *testing.T) {
	sets := rules.NewMemoryStore()
	down := errors.New("postgres is down")
	sets.FailWith(down)

	err := save_rule_set_usecase.New(sets, guard.Parts()).Execute(t.Context(), rush)

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, rules.ErrInvalidSet)
}
