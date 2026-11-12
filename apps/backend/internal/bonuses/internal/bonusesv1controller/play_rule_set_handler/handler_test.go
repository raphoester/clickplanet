package play_rule_set_handler_test

import (
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func booted(
	t *testing.T, kept ...cprules.Snapshot[*bonusesv1.Rules],
) (play_rule_set_handler.PlayRuleSetHandler, *cprules.MemoryChoices, *cprules.InForce[*bonusesv1.Rules]) {
	t.Helper()

	sets := cprules.NewMemoryStore(parts.All())
	for _, each := range append([]cprules.Snapshot[*bonusesv1.Rules]{rules.Seeded()}, kept...) {
		sets.Keep(each)
	}
	choices, inForce := cprules.NewMemoryChoices(), cprules.NewInForce[*bonusesv1.Rules](cptime.SystemClock{})
	useCase := cprules.NewPlayUseCase(sets, choices, inForce, nil)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	return play_rule_set_handler.New(useCase), choices, inForce
}

func play(t *testing.T, handler play_rule_set_handler.PlayRuleSetHandler, set string, decidedAt time.Time) error {
	t.Helper()

	_, err := handler.PlayRuleSet(t.Context(), connect.NewRequest(&bonusesv1.PlayRuleSetRequest{
		RuleSet: set, DecidedAt: timestamppb.New(decidedAt),
	}))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestTheSetNamedPlays(t *testing.T) {
	handler, choices, inForce := booted(t, rules.Named("rush", rules.AsIs))

	require.NoError(t, play(t, handler, "rush", finale))

	assert.Equal(t, cprules.SetName("rush"), inForce.Name())
	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.True(t, finale.Equal(kept.DecidedAt))
}

func TestASetTheOffersDoNotKeepIsAFailedPrecondition(t *testing.T) {
	handler, _, inForce := booted(t)

	err := play(t, handler, "rush", finale)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorIs(t, err, cprules.ErrUnknownSet)
	assert.Equal(t, cprules.Default, inForce.Name())
}

func TestASetThatCannotPlayIsAFailedPrecondition(t *testing.T) {
	handler, _, inForce := booted(t, rules.Named("rush", func(r *bonusesv1.Rules) { r.Spread.MaxPerBox = proto.Uint32(0) }))

	err := play(t, handler, "rush", finale)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Equal(t, cprules.Default, inForce.Name())
}

func TestAChoiceThatCannotBeKeptIsPassedOnForTheErrorNet(t *testing.T) {
	handler, choices, _ := booted(t, rules.Named("rush", rules.AsIs))
	down := errors.New("postgres is down")
	choices.FailWith(down)

	require.ErrorIs(t, play(t, handler, "rush", finale), down)
}
