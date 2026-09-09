package planetv3controller

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

func newTestClient(t *testing.T, svc stubService) planetv1connect.ClickServiceClient {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewClickServiceHandler(
		NewClickService(svc, stubChecker{}),
		connect.WithInterceptors(NewErrorInterceptor(nil)),
	))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return planetv1connect.NewClickServiceClient(server.Client(), server.URL)
}

func TestClick(t *testing.T) {
	click := func(client planetv1connect.ClickServiceClient, country string) error {
		_, err := client.Click(context.Background(),
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
		context.Background(), connect.NewRequest(&planetv1.MapDensityRequest{}))
	require.NoError(t, err)
	require.Equal(t, uint32(100), res.Msg.GetDensity())
}
