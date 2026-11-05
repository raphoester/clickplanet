package promov1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1/promov1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler"
)

type teaser struct{}

func (teaser) Teaser(context.Context, string) (*promov1.GetTeaserResponse, error) {
	return &promov1.GetTeaserResponse{Cards: []*promov1.TeaserCard{{Headline: "War"}}}, nil
}

func TestATeaserIsAGetAnyCacheMayKeepForAMinute(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(promov1connect.NewPromoServiceHandler(
		promov1controller.PromoService{GetTeaserHandler: get_teaser_handler.New(teaser{})},
		connect.WithInterceptors(promov1controller.NewCacheInterceptor()),
	))
	methods := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	client := promov1connect.NewPromoServiceClient(server.Client(), server.URL, connect.WithHTTPGet())
	res, err := client.GetTeaser(t.Context(), connect.NewRequest(&promov1.GetTeaserRequest{BattleId: "tonight"}))
	require.NoError(t, err)

	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=60, s-maxage=60", res.Header().Get("Cache-Control"))
	assert.Equal(t, "War", res.Msg.GetCards()[0].GetHeadline())
}
