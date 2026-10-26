package metagamev1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1/metagamev1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_turn_handler"
)

type battle struct{}

func (battle) Turn(context.Context) (*metagamev1.GetTurnResponse, error) {
	return &metagamev1.GetTurnResponse{Kind: metagamev1.EntryKind_ENTRY_KIND_BATTLE}, nil
}

func TestTheTurnIsAGetAnyCacheMayKeepForFiveSeconds(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(metagamev1connect.NewMetagameServiceHandler(
		metagamev1controller.MetagameService{GetTurnHandler: get_turn_handler.New(battle{})},
		connect.WithInterceptors(metagamev1controller.NewCacheInterceptor()),
	))
	methods := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	client := metagamev1connect.NewMetagameServiceClient(server.Client(), server.URL, connect.WithHTTPGet())
	res, err := client.GetTurn(t.Context(), connect.NewRequest(&metagamev1.GetTurnRequest{}))
	require.NoError(t, err)

	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=5, s-maxage=5", res.Header().Get("Cache-Control"))
	assert.Equal(t, metagamev1.EntryKind_ENTRY_KIND_BATTLE, res.Msg.GetKind())
}
