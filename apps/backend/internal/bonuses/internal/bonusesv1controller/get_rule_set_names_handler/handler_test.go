package get_rule_set_names_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonusesv1controller/get_rule_set_names_handler"
)

type stubQuery struct {
	names []string
	err   error
}

func (s stubQuery) Names(context.Context) ([]string, error) { return s.names, s.err }

func TestTheNamesAreTheQuerys(t *testing.T) {
	res, err := get_rule_set_names_handler.New(stubQuery{names: []string{"default", "rush"}}).
		GetRuleSetNames(t.Context(), connect.NewRequest(&bonusesv1.GetRuleSetNamesRequest{}))

	require.NoError(t, err)
	assert.Equal(t, []string{"default", "rush"}, res.Msg.GetNames())
}
