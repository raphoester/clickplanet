package gamemasterv1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_turn_handler"
)

type battle struct{}

func (battle) Turn(context.Context) (*gamemasterv1.Turn, error) {
	return &gamemasterv1.Turn{Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE}, nil
}

func TestTheTurnIsAGetAnyCacheMayKeepForFiveSeconds(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(gamemasterv1connect.NewGamemasterServiceHandler(
		gamemasterv1controller.GamemasterService{GetTurnHandler: get_turn_handler.New(battle{})},
		connect.WithInterceptors(gamemasterv1controller.NewCacheInterceptor()),
	))
	methods := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	client := gamemasterv1connect.NewGamemasterServiceClient(server.Client(), server.URL, connect.WithHTTPGet())
	res, err := client.GetTurn(t.Context(), connect.NewRequest(&gamemasterv1.GetTurnRequest{}))
	require.NoError(t, err)

	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=5, s-maxage=5", res.Header().Get("Cache-Control"))
	assert.Equal(t, gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, res.Msg.GetTurn().GetKind())
}
