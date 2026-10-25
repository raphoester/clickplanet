package override_rules_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/override_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/override_rules_usecase"
)

type stubUseCase struct {
	err   error
	asked []override_rules_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in override_rules_usecase.In) error {
	s.asked = append(s.asked, in)
	return s.err
}

var until = time.Date(2026, 10, 31, 21, 30, 0, 0, time.UTC)

func override(t *testing.T, useCase *stubUseCase, req *planetv1.OverrideRulesRequest) error {
	t.Helper()

	_, err := override_rules_handler.New(useCase).OverrideRules(t.Context(), connect.NewRequest(req))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestTheOverrideAskedIsPassedOn(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, override(t, useCase, &planetv1.OverrideRulesRequest{RuleSet: "rush", Until: timestamppb.New(until)}))

	assert.Equal(t, []override_rules_usecase.In{{Set: "rush", Until: until}}, useCase.asked)
}

func TestAnOverrideWithNoEndIsAnInvalidArgumentAndAsksNothing(t *testing.T) {
	useCase := &stubUseCase{}

	err := override(t, useCase, &planetv1.OverrideRulesRequest{RuleSet: "rush"})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}

func TestAnOverrideThatCannotRunIsAnInvalidArgument(t *testing.T) {
	for _, refusal := range []error{rules.ErrUnknownSet, rules.ErrInvalidOverride} {
		err := override(t, &stubUseCase{err: refusal}, &planetv1.OverrideRulesRequest{
			RuleSet: "rush", Until: timestamppb.New(until),
		})

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", refusal)
	}
}

func TestAFailureIsPassedOnForTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	err := override(t, &stubUseCase{err: down}, &planetv1.OverrideRulesRequest{RuleSet: "rush", Until: timestamppb.New(until)})

	require.ErrorIs(t, err, down)
}
