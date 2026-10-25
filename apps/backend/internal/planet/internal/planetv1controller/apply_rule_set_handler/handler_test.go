package apply_rule_set_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/apply_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/apply_rule_set_usecase"
)

type stubUseCase struct {
	err   error
	asked []apply_rule_set_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in apply_rule_set_usecase.In) error {
	s.asked = append(s.asked, in)
	return s.err
}

func apply(t *testing.T, handler apply_rule_set_handler.ApplyRuleSetHandler) error {
	t.Helper()

	_, err := handler.ApplyRuleSet(t.Context(), connect.NewRequest(&planetv1.ApplyRuleSetRequest{
		RequestId: "season-0:finale", RuleSet: "rush",
	}))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestTheRequestIsAppliedAsTheHandlersSource(t *testing.T) {
	for _, source := range []rules.Source{rules.SourceInternal, rules.SourceOperator} {
		useCase := &stubUseCase{}

		require.NoError(t, apply(t, apply_rule_set_handler.New(useCase, source)))

		assert.Equal(t, []apply_rule_set_usecase.In{{Request: "season-0:finale", Set: "rush", Source: source}}, useCase.asked)
	}
}

func TestARequestThatCannotBeAppliedIsAnInvalidArgument(t *testing.T) {
	for _, refusal := range []error{rules.ErrUnknownSet, rules.ErrNoRequest, rules.ErrLongRequest} {
		err := apply(t, apply_rule_set_handler.New(&stubUseCase{err: refusal}, rules.SourceOperator))

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", refusal)
	}
}

func TestAFailureIsPassedOnForTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	err := apply(t, apply_rule_set_handler.New(&stubUseCase{err: down}, rules.SourceInternal))

	require.ErrorIs(t, err, down)
}
