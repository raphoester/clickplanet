package rpc_antibot_rule_sets_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/rpc_antibot_rule_sets"
)

type stubAntibot struct {
	antibotv1connect.UnimplementedInternalServiceHandler

	names []string
	err   error
}

func (s stubAntibot) GetRuleSetNames(
	context.Context,
	*connect.Request[antibotv1.GetRuleSetNamesRequest],
) (*connect.Response[antibotv1.GetRuleSetNamesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&antibotv1.GetRuleSetNamesResponse{Names: s.names}), nil
}

func ruleSets(t *testing.T, mux *http.ServeMux) *rpc_antibot_rule_sets.RuleSets {
	t.Helper()

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_antibot_rule_sets.New(antibotv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func serving(antibot stubAntibot) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(antibotv1connect.NewInternalServiceHandler(antibot))
	return mux
}

func TestASetTheAntibotKeepsIsKept(t *testing.T) {
	require.NoError(t, ruleSets(t, serving(stubAntibot{names: []string{"default", "rush"}})).Keeps(t.Context(), "rush"))
}

func TestASetTheAntibotDoesNotKeepIsUnknownToTheAntibot(t *testing.T) {
	err := ruleSets(t, serving(stubAntibot{names: []string{"default"}})).Keeps(t.Context(), "rush")

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Equal(t, `the antibot keeps no rule set of this name: "rush"`, err.Error())
}

func TestAnAntibotThatIsOffLacksNoSet(t *testing.T) {
	require.NoError(t, ruleSets(t, http.NewServeMux()).Keeps(t.Context(), "rush"))
}

func TestAnAntibotThatFailsIsAnError(t *testing.T) {
	err := ruleSets(t, serving(stubAntibot{err: connect.NewError(connect.CodeInternal, errors.New("boom"))})).
		Keeps(t.Context(), "rush")

	require.ErrorContains(t, err, "failed to ask the antibot")
}
