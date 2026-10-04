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
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/rpc_take_feed"
)

type stubPlanet struct {
	planetv1connect.UnimplementedInternalServiceHandler

	start   uint64
	entries []*planetv1.LogEntry
	err     error
	asked   chan *planetv1.ReadLogRequest
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
	_ context.Context, req *connect.Request[planetv1.ReadLogRequest],
) (*connect.Response[planetv1.ReadLogResponse], error) {
	s.asked <- req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&planetv1.ReadLogResponse{Entries: s.entries}), nil
}

func take(position uint64, take *planetv1.Take) *planetv1.LogEntry {
	return &planetv1.LogEntry{Position: position, Fact: &planetv1.LogEntry_Take{Take: take}}
}

type dialer struct {
	client connect.HTTPClient
	url    string
	err    error
}

func (d dialer) Dial() (connect.HTTPClient, string, error) {
	return d.client, d.url, d.err
}

const adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var noon = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

func feed(t *testing.T, planet stubPlanet) *rpc_take_feed.Feed {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewInternalServiceHandler(planet))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return rpc_take_feed.New(dialer{client: server.Client(), url: server.URL})
}

func TestTheStartIsPlanets(t *testing.T) {
	start, err := feed(t, stubPlanet{start: 42}).Start(t.Context())

	require.NoError(t, err)
	assert.Equal(t, takes.Position(42), start)
}

func TestABatchIsPlanetsTakesFromThePositionAsked(t *testing.T) {
	asked := make(chan *planetv1.ReadLogRequest, 1)
	planet := stubPlanet{asked: asked, entries: []*planetv1.LogEntry{
		take(7, &planetv1.Take{TileId: 1, AccountId: adaID, Country: "fr", TakenAt: timestamppb.New(noon)}),
		take(8, &planetv1.Take{TileId: 2, AccountId: adaID, Country: "", TakenAt: timestamppb.New(noon)}),
		take(9, &planetv1.Take{TileId: 3, Country: "fr", TakenAt: timestamppb.New(noon)}),
		take(11, &planetv1.Take{TileId: 4, AccountId: adaID, Country: "fr", TakenAt: timestamppb.New(noon), Reverted: true}),
	}}

	batch, err := feed(t, planet).Batch(t.Context(), 7)

	require.NoError(t, err)
	request := <-asked
	assert.Equal(t, uint64(7), request.GetFromPosition())
	assert.Positive(t, request.GetLimit())
	assert.Equal(t, takes.Position(7), batch.From())
	assert.Equal(t, takes.Position(12), batch.Next())
	ada, err := players.AccountIDOf(adaID)
	require.NoError(t, err)
	assert.Equal(t, []players.Stats{players.NewStats(ada).WithTake(noon)}, batch.Tallied(map[players.AccountID]players.Stats{}),
		"a clear, a take with no account and a reverted take count nothing")
}

func TestAnEntryOfAKindPlayerDoesNotKnowIsSkippedAndReadPast(t *testing.T) {
	asked := make(chan *planetv1.ReadLogRequest, 1)

	batch, err := feed(t, stubPlanet{asked: asked, entries: []*planetv1.LogEntry{
		{Position: 3},
		take(4, &planetv1.Take{TileId: 1, AccountId: adaID, Country: "fr", TakenAt: timestamppb.New(noon)}),
		{Position: 6},
	}}).Batch(t.Context(), 3)

	require.NoError(t, err)
	assert.Equal(t, 1, batch.Len())
	assert.Equal(t, takes.Position(7), batch.Next(), "the position moves past the last entry read, whatever its kind")
}

func TestATakeWithNoAccountIsNobodys(t *testing.T) {
	asked := make(chan *planetv1.ReadLogRequest, 1)

	batch, err := feed(t, stubPlanet{asked: asked, entries: []*planetv1.LogEntry{
		take(0, &planetv1.Take{Country: "fr", TakenAt: timestamppb.New(noon)}),
	}}).Batch(t.Context(), 0)

	require.NoError(t, err)
	assert.Empty(t, batch.Accounts())
	assert.Equal(t, takes.Position(1), batch.Next())
}

func TestAnAnswerThatIsNotATakeIsRefused(t *testing.T) {
	for name, entries := range map[string][]*planetv1.LogEntry{
		"an id that is no account":          {take(4, &planetv1.Take{AccountId: "not-an-account", Country: "fr", TakenAt: timestamppb.New(noon)})},
		"no time":                           {take(4, &planetv1.Take{AccountId: adaID, Country: "fr"})},
		"before the position asked":         {take(3, &planetv1.Take{AccountId: adaID, Country: "fr", TakenAt: timestamppb.New(noon)})},
		"an entry of any kind out of order": {{Position: 6}, {Position: 5}},
	} {
		asked := make(chan *planetv1.ReadLogRequest, 1)

		_, err := feed(t, stubPlanet{asked: asked, entries: entries}).Batch(t.Context(), 4)

		assert.Error(t, err, name)
	}
}

func TestPlanetFailingOrUnreachableIsAnError(t *testing.T) {
	failing := stubPlanet{asked: make(chan *planetv1.ReadLogRequest, 1), err: connect.NewError(connect.CodeInternal, errors.New("down"))}
	_, err := feed(t, failing).Batch(t.Context(), 0)
	require.Error(t, err)
	_, err = feed(t, failing).Start(t.Context())
	require.Error(t, err)

	unreachable := rpc_take_feed.New(dialer{err: errors.New("no internal listener")})
	_, err = unreachable.Batch(t.Context(), 0)
	require.Error(t, err)
	_, err = unreachable.Start(t.Context())
	require.Error(t, err)
}
