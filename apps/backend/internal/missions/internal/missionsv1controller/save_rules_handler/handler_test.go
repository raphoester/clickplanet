package save_rules_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/save_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
)

type stubUseCase struct {
	asked []*missionsv1.Rules
	err   error
}

func (s *stubUseCase) Execute(_ context.Context, in *missionsv1.Rules) error {
	s.asked = append(s.asked, in)
	return s.err
}

var in = &missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{GoalPercent: 40, LeastGoal: 300, FewestPoints: 2, MostPoints: 8}}

func TestTheOperatorsNumbersAreSaved(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := save_rules_handler.New(useCase).SaveRules(t.Context(), connect.NewRequest(&missionsv1.SaveRulesRequest{Rules: in}))

	require.NoError(t, err)
	require.Len(t, useCase.asked, 1)
	assert.True(t, proto.Equal(in, useCase.asked[0]))
}

func TestNumbersThatCannotPlayAreTheCallersFault(t *testing.T) {
	useCase := &stubUseCase{err: fmt.Errorf("%w: flagMissions: no", rules.ErrInvalidRules)}

	_, err := save_rules_handler.New(useCase).SaveRules(t.Context(), connect.NewRequest(&missionsv1.SaveRulesRequest{Rules: in}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAStoreThatFailsIsNotTheCallersFault(t *testing.T) {
	useCase := &stubUseCase{err: errors.New("postgres is gone")}

	_, err := save_rules_handler.New(useCase).SaveRules(t.Context(), connect.NewRequest(&missionsv1.SaveRulesRequest{Rules: in}))

	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
