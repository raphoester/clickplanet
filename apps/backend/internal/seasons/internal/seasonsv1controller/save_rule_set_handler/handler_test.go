package save_rule_set_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules/usecases/save_rule_set_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/save_rule_set_handler"
)

type stubUseCase struct {
	in  rules.Snapshot
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in rules.Snapshot) error {
	s.in = in
	return s.err
}

var rush = rules.Battle("rush", 3)

func save(t *testing.T, useCase *stubUseCase) error {
	t.Helper()
	_, err := save_rule_set_handler.New(useCase).SaveRuleSet(t.Context(), connect.NewRequest(&seasonsv1.SaveRuleSetRequest{
		Set: &seasonsv1.RuleSet{Name: "rush", Rules: rush.Rules},
	}))
	return err //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheSetReachesTheUseCaseAsWritten(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, save(t, useCase))

	assert.Equal(t, rush.Name, useCase.in.Name)
	assert.True(t, proto.Equal(rush.Rules, useCase.in.Rules))
}

func TestASetThatCannotPlayIsAnInvalidArgumentThatSaysWhy(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("cannot keep it: %w: ownRound.times is 0", rules.ErrInvalidSet)}

	err := save(t, useCase)

	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "ownRound.times is 0")
}

func TestASetThatPlaysNowIsAFailedPrecondition(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("cannot keep it: %w", save_rule_set_usecase.ErrPlaying)}

	err := save(t, useCase)

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "this rule set plays now")
}

func TestASaveThatFailsFailsTheCall(t *testing.T) {
	down := errors.New("postgres is down")

	err := save(t, &stubUseCase{err: down})

	require.ErrorIs(t, err, down)
}
