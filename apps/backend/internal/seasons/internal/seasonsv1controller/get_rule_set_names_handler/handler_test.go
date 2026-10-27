package get_rule_set_names_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_rule_set_names_handler"
)

type stubQuery struct {
	names []string
	err   error
}

func (s stubQuery) Names(context.Context) ([]string, error) { return s.names, s.err }

func TestTheNamesAreTheQuerys(t *testing.T) {
	res, err := get_rule_set_names_handler.New(stubQuery{names: []string{"default", "rush"}}).
		GetRuleSetNames(t.Context(), connect.NewRequest(&seasonsv1.GetRuleSetNamesRequest{}))

	require.NoError(t, err)
	assert.Equal(t, []string{"default", "rush"}, res.Msg.GetNames())
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_rule_set_names_handler.New(stubQuery{err: down}).
		GetRuleSetNames(t.Context(), connect.NewRequest(&seasonsv1.GetRuleSetNamesRequest{}))

	require.ErrorIs(t, err, down)
}
