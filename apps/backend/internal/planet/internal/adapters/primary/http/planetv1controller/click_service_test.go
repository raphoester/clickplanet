package planetv1controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain"
	"github.com/stretchr/testify/require"
)

type stubService struct{ err error }

func (s stubService) HandleClick(context.Context, uint32, string) error { return s.err }

type stubChecker struct{}

func (stubChecker) CheckTile(uint32) bool { return true }
func (stubChecker) MaxIndex() uint32      { return 100 }

type stubSubscriber struct {
	updates chan domain.TileUpdate
	err     error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan domain.TileUpdate, error) {
	return s.updates, s.err
}

func newTestClient(t *testing.T, svc stubService) planetv1connect.ClickServiceClient {
	t.Helper()
	return newTestClientWith(t, svc, stubSubscriber{}, DefaultHeartbeat)
}

func newTestClientWith(
	t *testing.T,
	svc stubService,
	subscriber UpdatesSubscriber,
	heartbeat time.Duration,
) planetv1connect.ClickServiceClient {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewClickServiceHandler(
		NewClickService(svc, stubChecker{}, stubMapReader{}, subscriber, heartbeat, nil, nil, nil, nil),
		connect.WithInterceptors(NewErrorInterceptor(nil)),
	))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return planetv1connect.NewClickServiceClient(server.Client(), server.URL)
}

func TestClick(t *testing.T) {
	click := func(client planetv1connect.ClickServiceClient, country string) error {
		_, err := client.Click(t.Context(),
			connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: country}))
		return err
	}

	t.Run("a valid click answers empty", func(t *testing.T) {
		require.NoError(t, click(newTestClient(t, stubService{}), "fr"))
	})

	t.Run("a caller error becomes invalid_argument", func(t *testing.T) {
		client := newTestClient(t, stubService{err: fmt.Errorf("%w: nope", domain.ErrInvalidArgument)})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(click(client, "zz")))
	})

	t.Run("any other failure stays internal and does not leak the cause", func(t *testing.T) {
		client := newTestClient(t, stubService{err: fmt.Errorf("disk on fire")})
		err := click(client, "fr")
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.NotContains(t, err.Error(), "disk on fire")
	})
}

func TestMapDensity(t *testing.T) {
	res, err := newTestClient(t, stubService{}).MapDensity(
		t.Context(), connect.NewRequest(&planetv1.MapDensityRequest{}))
	require.NoError(t, err)
	require.Equal(t, uint32(100), res.Msg.GetDensity())
}

func TestListenForEvents(t *testing.T) {
	// The call blocks until the first frame, so subscriptions are seeded before it.
	listen := func(
		t *testing.T,
		subscriber UpdatesSubscriber,
		heartbeat time.Duration,
	) (*connect.ServerStreamForClient[planetv1.PlanetEvent], error) {
		t.Helper()

		stream, err := newTestClientWith(t, stubService{}, subscriber, heartbeat).ListenForEvents(
			t.Context(), connect.NewRequest(&planetv1.ListenForEventsRequest{}))

		// Closing it releases the handler, which is still parked on its
		// subscription; httptest.Server.Close blocks forever otherwise.
		if stream != nil {
			t.Cleanup(func() { _ = stream.Close() })
		}

		return stream, err
	}

	t.Run("carries an update to the caller", func(t *testing.T) {
		updates := make(chan domain.TileUpdate, 1)
		updates <- domain.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}

		stream, err := listen(t, stubSubscriber{updates: updates}, DefaultHeartbeat)
		require.NoError(t, err)

		require.True(t, stream.Receive())
		update := stream.Msg().GetTileUpdate()
		require.NotNil(t, update, "a tile update arrives as the tile_update case")
		require.Equal(t, uint32(42), update.GetTileId())
		require.Equal(t, "fr", update.GetCountryId())
		require.Equal(t, "de", update.GetPreviousCountryId())
	})

	t.Run("keeps a silent stream alive with heartbeats", func(t *testing.T) {
		// Nothing is ever published, so every frame here is a heartbeat.
		stream, err := listen(t, stubSubscriber{updates: make(chan domain.TileUpdate)}, 10*time.Millisecond)
		require.NoError(t, err)

		for i := range 3 {
			require.Truef(t, stream.Receive(), "heartbeat %d never arrived", i)
			require.NotNil(t, stream.Msg().GetHeartbeat(), "frame %d is not a heartbeat", i)
			require.Nil(t, stream.Msg().GetTileUpdate())
		}
	})

	t.Run("ends when the subscription closes", func(t *testing.T) {
		updates := make(chan domain.TileUpdate)
		close(updates)

		stream, err := listen(t, stubSubscriber{updates: updates}, DefaultHeartbeat)
		require.NoError(t, err)

		require.False(t, stream.Receive())
		require.NoError(t, stream.Err())
	})

	t.Run("a failed subscription stays internal and does not leak the cause", func(t *testing.T) {
		stream, err := listen(t, stubSubscriber{err: fmt.Errorf("disk on fire")}, DefaultHeartbeat)
		if err == nil {
			require.False(t, stream.Receive())
			err = stream.Err()
		}

		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.NotContains(t, err.Error(), "disk on fire")
	})
}

type stubMapReader struct{}

func (stubMapReader) StateBatchDense(start uint32, end uint32) (domain.DenseBatch, error) {
	if start > end {
		return domain.DenseBatch{}, fmt.Errorf("invalid tile range")
	}

	return domain.DenseBatch{
		Start: start,
		Codes: []string{"", "fr"},
		Tiles: []byte{0x01, 0x00, 0x00, 0x00},
	}, nil
}

func TestGetMap(t *testing.T) {
	getMap := func(t *testing.T, req *planetv1.GetMapRequest) (*connect.Response[planetv1.GetMapResponse], error) {
		t.Helper()
		return newTestClient(t, stubService{}).GetMap(t.Context(), connect.NewRequest(req))
	}

	t.Run("answers the dense batch with its code table", func(t *testing.T) {
		res, err := getMap(t, &planetv1.GetMapRequest{StartTileId: 7, EndTileId: 8})
		require.NoError(t, err)
		require.Equal(t, uint32(7), res.Msg.GetStartTileId())
		require.Equal(t, []string{"", "fr"}, res.Msg.GetCodes())
		require.Equal(t, []byte{0x01, 0x00, 0x00, 0x00}, res.Msg.GetTiles())
	})

	t.Run("is cacheable", func(t *testing.T) {
		res, err := getMap(t, &planetv1.GetMapRequest{})
		require.NoError(t, err)
		require.Equal(t, "public, max-age=5", res.Header().Get("Cache-Control"))
	})

	t.Run("an inverted range is a caller error", func(t *testing.T) {
		_, err := getMap(t, &planetv1.GetMapRequest{StartTileId: 9, EndTileId: 2})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}
