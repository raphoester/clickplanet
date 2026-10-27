package rpc_seasons_rule_sets_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/rpc_seasons_rule_sets"
)

type stubSeasons struct {
	seasonsv1connect.UnimplementedInternalServiceHandler

	names []string
	err   error
}

func (s stubSeasons) GetRuleSetNames(
	context.Context,
	*connect.Request[seasonsv1.GetRuleSetNamesRequest],
) (*connect.Response[seasonsv1.GetRuleSetNamesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&seasonsv1.GetRuleSetNamesResponse{Names: s.names}), nil
}

func ruleSets(t *testing.T, seasons stubSeasons) *rpc_seasons_rule_sets.RuleSets {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(seasonsv1connect.NewInternalServiceHandler(seasons))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_seasons_rule_sets.New(seasonsv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestASetSeasonsKeepsIsKept(t *testing.T) {
	require.NoError(t, ruleSets(t, stubSeasons{names: []string{"default", "rush"}}).Keeps(t.Context(), "rush"))
}

func TestASetSeasonsDoesNotKeepIsUnknown(t *testing.T) {
	err := ruleSets(t, stubSeasons{names: []string{"default"}}).Keeps(t.Context(), "rush")

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Equal(t, `seasons keeps no rule set of this name: "rush"`, err.Error())
}

func TestASeasonsModuleThatFailsIsAnError(t *testing.T) {
	err := ruleSets(t, stubSeasons{err: connect.NewError(connect.CodeInternal, errors.New("boom"))}).Keeps(t.Context(), "rush")

	require.ErrorContains(t, err, "failed to ask the seasons module")
	assert.NotErrorIs(t, err, schedule.ErrUnknownSet)
}
