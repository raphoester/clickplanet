package rpc_take_feed_test

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
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/rpc_take_feed"
)

type stubPlanet struct {
	planetv1connect.UnimplementedInternalServiceHandler

	start   uint64
	entries []*planetv1.LogEntry
	err     error
}

func (s stubPlanet) GetFeedStart(
	context.Context, *connect.Request[planetv1.GetFeedStartRequest],
) (*connect.Response[planetv1.GetFeedStartResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&planetv1.GetFeedStartResponse{Position: s.start}), nil
}

func (s stubPlanet) ReadLog(
	context.Context, *connect.Request[planetv1.ReadLogRequest],
) (*connect.Response[planetv1.ReadLogResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&planetv1.ReadLogResponse{Entries: s.entries}), nil
}

func take(position uint64, take *planetv1.Take) *planetv1.LogEntry {
	return &planetv1.LogEntry{Position: position, Fact: &planetv1.LogEntry_Take{Take: take}}
}

const adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var noon = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

func feed(t *testing.T, planet stubPlanet) *rpc_take_feed.Feed {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewInternalServiceHandler(planet))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_take_feed.New(planetv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func TestTheStartIsPlanets(t *testing.T) {
	start, err := feed(t, stubPlanet{start: 42}).Start(t.Context())

	require.NoError(t, err)
	assert.Equal(t, standings.Position(42), start)
}

func TestTheTakesThatCountAreKeptAndAnEntryOfAnUnknownKindIsReadPast(t *testing.T) {
	batch, err := feed(t, stubPlanet{entries: []*planetv1.LogEntry{
		{Position: 3},
		take(4, &planetv1.Take{AccountId: adaID, Country: "fr", TakenAt: timestamppb.New(noon)}),
		take(5, &planetv1.Take{Country: "fr", TakenAt: timestamppb.New(noon)}),
		take(6, &planetv1.Take{AccountId: adaID, Country: "de", TakenAt: timestamppb.New(noon), Reverted: true}),
		{Position: 8},
	}}).Batch(t.Context(), 3)

	require.NoError(t, err)
	ada, err := standings.AccountIDOf(adaID)
	require.NoError(t, err)
	assert.Equal(t, []standings.Take{{Account: ada, Country: "fr", At: noon}}, batch.Takes())
	assert.Equal(t, standings.Position(9), batch.Next(), "the position moves past the last entry read, whatever its kind")
}

func TestAnAnswerThatIsNotATakeIsRefused(t *testing.T) {
	for name, entries := range map[string][]*planetv1.LogEntry{
		"an id that is no account":          {take(4, &planetv1.Take{AccountId: "not-an-account", Country: "fr", TakenAt: timestamppb.New(noon)})},
		"no time":                           {take(4, &planetv1.Take{AccountId: adaID, Country: "fr"})},
		"before the position asked":         {take(3, &planetv1.Take{AccountId: adaID, Country: "fr", TakenAt: timestamppb.New(noon)})},
		"an entry of any kind out of order": {{Position: 6}, {Position: 5}},
	} {
		_, err := feed(t, stubPlanet{entries: entries}).Batch(t.Context(), 4)

		assert.Error(t, err, name)
	}
}

func TestAPlanetThatFailsIsAnError(t *testing.T) {
	failing := stubPlanet{err: connect.NewError(connect.CodeInternal, errors.New("down"))}

	_, err := feed(t, failing).Batch(t.Context(), 0)
	require.Error(t, err)
	_, err = feed(t, failing).Start(t.Context())
	require.Error(t, err)
}
