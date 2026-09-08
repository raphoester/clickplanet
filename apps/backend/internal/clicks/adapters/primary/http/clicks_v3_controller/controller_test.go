package clicks_v3_controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/stretchr/testify/require"
)

type stubService struct{ err error }

func (s stubService) HandleClick(context.Context, uint32, string) error { return s.err }

type stubChecker struct{}

func (stubChecker) CheckTile(uint32) bool { return true }
func (stubChecker) MaxIndex() uint32      { return 100 }

type stubEncoder struct{ body []byte }

func (s stubEncoder) EncodeStateBatch(start uint32, end uint32) ([]byte, error) {
	if start > end {
		return nil, fmt.Errorf("invalid range")
	}
	return s.body, nil
}

func newTestServer(t *testing.T, svc stubService) (*httptest.Server, planetv1connect.ClickServiceClient) {
	t.Helper()

	mux := http.NewServeMux()
	New(svc, stubChecker{}, stubEncoder{body: []byte("CPM1")}, nil).DeclareRoutes(mux)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server, planetv1connect.NewClickServiceClient(server.Client(), server.URL)
}

func TestClick(t *testing.T) {
	t.Run("a valid click answers empty", func(t *testing.T) {
		_, client := newTestServer(t, stubService{})
		_, err := client.Click(context.Background(),
			connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"}))
		require.NoError(t, err)
	})

	t.Run("a caller error becomes invalid_argument", func(t *testing.T) {
		_, client := newTestServer(t, stubService{err: fmt.Errorf("%w: nope", domain.ErrInvalidArgument)})
		_, err := client.Click(context.Background(),
			connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "zz"}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("any other failure stays internal and does not leak the cause", func(t *testing.T) {
		_, client := newTestServer(t, stubService{err: fmt.Errorf("disk on fire")})
		_, err := client.Click(context.Background(),
			connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.NotContains(t, err.Error(), "disk on fire")
	})
}

func TestMapDensity(t *testing.T) {
	_, client := newTestServer(t, stubService{})
	res, err := client.MapDensity(context.Background(), connect.NewRequest(&planetv1.MapDensityRequest{}))
	require.NoError(t, err)
	require.Equal(t, uint32(100), res.Msg.GetDensity())
}

func TestGetMap(t *testing.T) {
	t.Run("answers the encoded chunk, cacheable", func(t *testing.T) {
		server, _ := newTestServer(t, stubService{})
		res, err := server.Client().Get(server.URL + "/map")
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, "application/octet-stream", res.Header.Get("Content-Type"))
		require.Equal(t, "public, max-age=5", res.Header.Get("Cache-Control"))
	})

	t.Run("a range that is not a number is refused", func(t *testing.T) {
		server, _ := newTestServer(t, stubService{})
		res, err := server.Client().Get(server.URL + "/map?start=abc")
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("an inverted range is refused", func(t *testing.T) {
		server, _ := newTestServer(t, stubService{})
		res, err := server.Client().Get(server.URL + "/map?start=9&end=2")
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}
