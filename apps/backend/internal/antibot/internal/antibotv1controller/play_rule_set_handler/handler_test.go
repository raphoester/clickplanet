package play_rule_set_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func booted(
	t *testing.T, kept ...cprules.Snapshot[*antibotv1.Rules],
) (play_rule_set_handler.PlayRuleSetHandler, *cprules.MemoryChoices, *cprules.InForce[*antibotv1.Rules]) {
	t.Helper()

	choices, inForce := cprules.NewMemoryChoices(), cprules.NewInForce[*antibotv1.Rules](cptime.SystemClock{})
	sets := cprules.NewMemoryStore(guard.Parts(), append([]cprules.Snapshot[*antibotv1.Rules]{rules.Seeded()}, kept...)...)
	useCase := cprules.NewPlayUseCase(sets, choices, inForce, restated)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	return play_rule_set_handler.New(useCase), choices, inForce
}

func play(t *testing.T, handler play_rule_set_handler.PlayRuleSetHandler, set string, decidedAt time.Time) error {
	t.Helper()

	_, err := handler.PlayRuleSet(t.Context(), connect.NewRequest(&antibotv1.PlayRuleSetRequest{
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

func TestASetTheAntibotDoesNotKeepIsAFailedPrecondition(t *testing.T) {
	handler, _, inForce := booted(t)

	err := play(t, handler, "rush", finale)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorIs(t, err, cprules.ErrUnknownSet)
	assert.Equal(t, cprules.Default, inForce.Name())
}

func TestASetThatCannotPlayIsAFailedPrecondition(t *testing.T) {
	handler, _, inForce := booted(t,
		rules.Named("rush", func(r *antibotv1.Rules) { r.Jury.MinSuspects = proto.Uint32(0) }))

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

func restated(context.Context, cprules.SetName) error { return nil }
