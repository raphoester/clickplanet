package save_rule_set_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/save_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type stubUseCase struct {
	err   error
	asked []rules.Snapshot
}

func (s *stubUseCase) Execute(_ context.Context, in rules.Snapshot) error {
	s.asked = append(s.asked, in)
	return s.err
}

var rush = rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = proto.Float64(3) })

func save(t *testing.T, useCase *stubUseCase) error {
	t.Helper()

	_, err := save_rule_set_handler.New(useCase).SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{
		Set: &planetv1.RuleSet{Name: "rush", Rules: rush.Rules},
	}))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestTheSetIsSavedAsSent(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, save(t, useCase))

	require.Len(t, useCase.asked, 1)
	assert.Equal(t, rules.SetName("rush"), useCase.asked[0].Name)
	assert.True(t, proto.Equal(rush.Rules, useCase.asked[0].Rules))
}

func TestASetThatCannotPlayIsAnInvalidArgument(t *testing.T) {
	err := save(t, &stubUseCase{err: rules.ErrInvalidSet})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAFailureIsPassedOnForTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	require.ErrorIs(t, save(t, &stubUseCase{err: down}), down)
}
