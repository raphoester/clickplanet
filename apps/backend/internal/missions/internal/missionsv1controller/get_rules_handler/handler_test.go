package get_rules_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/get_rules_handler"
)

type stubQuery struct{ answer *missionsv1.GetRulesResponse }

func (s stubQuery) Rules(context.Context) (*missionsv1.GetRulesResponse, error) { return s.answer, nil }

func TestTheOperatorReadsTheRulesUnstored(t *testing.T) {
	answer := &missionsv1.GetRulesResponse{Rules: &missionsv1.Rules{FlagMissions: &missionsv1.FlagMissionRules{GoalPercent: 30}}}

	res, err := get_rules_handler.New(stubQuery{answer: answer}).GetRules(t.Context(), connect.NewRequest(&missionsv1.GetRulesRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(answer, res.Msg))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}
