package rpc_planet_rules_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale/rpc_planet_rules"
)

type stubPlanet struct {
	planetv1connect.UnimplementedInternalServiceHandler

	seen []*planetv1.SetRulesRequest
	err  error
}

func (s *stubPlanet) SetRules(
	_ context.Context,
	req *connect.Request[planetv1.SetRulesRequest],
) (*connect.Response[planetv1.SetRulesResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	s.seen = append(s.seen, req.Msg)
	return connect.NewResponse(&planetv1.SetRulesResponse{}), nil
}

func serve(t *testing.T, planet *stubPlanet) planetv1connect.InternalServiceClient {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewInternalServiceHandler(planet))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return planetv1connect.NewInternalServiceClient(server.Client(), server.URL)
}

var (
	finaleStarts = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonEnds   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
)

func switchesAt(at time.Time) finale.Switches {
	seasons := calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour}}})
	return finale.PhaseAt(seasons, at).Switches(finale.NewRules(finale.Config{RefillMultiplier: 3, BoxInterval: 2 * time.Minute}))
}

func TestTheFinalesSwitchesReachPlanetInItsTerms(t *testing.T) {
	planet := &stubPlanet{}

	require.NoError(t, rpc_planet_rules.New(serve(t, planet)).Set(t.Context(), switchesAt(finaleStarts)))

	require.Len(t, planet.seen, 1)
	assert.True(t, proto.Equal(&planetv1.SetRulesRequest{
		RefillMultiplier: 3,
		BoxIntervalMs:    120_000,
		Gift:             &planetv1.Gift{Tag: "season-0-finale", AccountsMadeBeforeUnixMs: finaleStarts.UnixMilli()},
	}, planet.seen[0]))
}

func TestTheEndFreezesTheMapAndGivesNothing(t *testing.T) {
	planet := &stubPlanet{}

	require.NoError(t, rpc_planet_rules.New(serve(t, planet)).Set(t.Context(), switchesAt(seasonEnds)))

	require.Len(t, planet.seen, 1)
	assert.True(t, proto.Equal(&planetv1.SetRulesRequest{Frozen: true}, planet.seen[0]))
}

func TestAFailureToAskIsAnError(t *testing.T) {
	planet := &stubPlanet{err: connect.NewError(connect.CodeInvalidArgument, errors.New("no"))}

	require.Error(t, rpc_planet_rules.New(serve(t, planet)).Set(t.Context(), switchesAt(seasonEnds)))
}
