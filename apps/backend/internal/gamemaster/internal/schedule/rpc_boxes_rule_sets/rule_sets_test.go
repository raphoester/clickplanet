package rpc_boxes_rule_sets_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1/boxesv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/rpc_boxes_rule_sets"
)

type stubBoxes struct {
	boxesv1connect.UnimplementedInternalServiceHandler

	names []string
	err   error
}

func (s stubBoxes) GetRuleSetNames(
	context.Context, *connect.Request[boxesv1.GetRuleSetNamesRequest],
) (*connect.Response[boxesv1.GetRuleSetNamesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&boxesv1.GetRuleSetNamesResponse{Names: s.names}), nil
}

func ruleSets(t *testing.T, boxes stubBoxes) *rpc_boxes_rule_sets.RuleSets {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(boxesv1connect.NewInternalServiceHandler(boxes))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_boxes_rule_sets.New(boxesv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestASetTheBoxesKeepIsKept(t *testing.T) {
	require.NoError(t, ruleSets(t, stubBoxes{names: []string{"default", "rush"}}).Keeps(t.Context(), "rush"))
}

func TestASetTheBoxesDoNotKeepIsUnknownToThem(t *testing.T) {
	err := ruleSets(t, stubBoxes{names: []string{"default"}}).Keeps(t.Context(), "rush")

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Equal(t, `the boxes module keeps no rule set of this name: "rush"`, err.Error())
}

func TestBoxesThatFailToAnswerAreAnError(t *testing.T) {
	err := ruleSets(t, stubBoxes{err: connect.NewError(connect.CodeInternal, errors.New("boom"))}).
		Keeps(t.Context(), "rush")

	require.ErrorContains(t, err, "failed to ask the boxes module")
}
