package rpc_quizzes_rule_sets_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1/quizzesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_quizzes_rule_sets"
)

type stubQuizzes struct {
	quizzesv1connect.UnimplementedInternalServiceHandler

	names []string
	err   error
}

func (s stubQuizzes) GetRuleSetNames(
	context.Context, *connect.Request[quizzesv1.GetRuleSetNamesRequest],
) (*connect.Response[quizzesv1.GetRuleSetNamesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&quizzesv1.GetRuleSetNamesResponse{Names: s.names}), nil
}

func ruleSets(t *testing.T, quizzes stubQuizzes) *rpc_quizzes_rule_sets.RuleSets {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(quizzesv1connect.NewInternalServiceHandler(quizzes))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_quizzes_rule_sets.New(quizzesv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestASetTheOffersKeepIsKept(t *testing.T) {
	require.NoError(t, ruleSets(t, stubQuizzes{names: []string{"default", "rush"}}).Keeps(t.Context(), "rush"))
}

func TestASetTheOffersDoNotKeepIsUnknownToThem(t *testing.T) {
	err := ruleSets(t, stubQuizzes{names: []string{"default"}}).Keeps(t.Context(), "rush")

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Equal(t, `the quizzes module keeps no rule set of this name: "rush"`, err.Error())
}

func TestOffersThatFailToAnswerAreAnError(t *testing.T) {
	err := ruleSets(t, stubQuizzes{err: connect.NewError(connect.CodeInternal, errors.New("boom"))}).
		Keeps(t.Context(), "rush")

	require.ErrorContains(t, err, "failed to ask the quizzes module")
}
