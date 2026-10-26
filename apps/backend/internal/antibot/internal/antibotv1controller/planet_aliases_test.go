package antibotv1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/ban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/inspect_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/antibotv1controller/unban_player_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/ban_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/inspect_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/unban_player_usecase"
)

var until = time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC)

type bans struct{ asked ban_player_usecase.In }

func (b *bans) Execute(_ context.Context, in ban_player_usecase.In) (ban_player_usecase.Out, error) {
	b.asked = in
	return ban_player_usecase.Out{Scope: "203.0.113.7", Offence: 2, Until: until, Enforced: true}, nil
}

type unbans struct{}

func (unbans) Execute(context.Context, unban_player_usecase.In) (unban_player_usecase.Out, error) {
	return unban_player_usecase.Out{Scope: "203.0.113.7", Offence: 2, Until: until}, nil
}

type inspections struct{}

func (inspections) Execute(_ context.Context, in inspect_player_usecase.In) (guard.Examination, error) {
	return guard.Examination{Scope: "203.0.113.7", Account: in.Account, Clicks: 42, Readings: []guard.Reading{{Watchdog: "metronome", Level: "suspect"}}}, nil
}

func serve(t *testing.T, service antibotv1controller.AdminService) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewAdminServiceHandler(planetv1connect.UnimplementedAdminServiceHandler{}))
	for _, build := range antibotv1controller.PlanetAliases(service) {
		mux.Handle(build())
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL
}

func TestTheOldPlanetCallsAreAnsweredByTheAntibot(t *testing.T) {
	banned := &bans{}
	url := serve(t, antibotv1controller.AdminService{
		BanPlayerHandler:     ban_player_handler.New(banned),
		UnbanPlayerHandler:   unban_player_handler.New(unbans{}),
		InspectPlayerHandler: inspect_player_handler.New(inspections{}),
	})

	for name, options := range map[string][]connect.ClientOption{"json": {connect.WithProtoJSON()}, "binary": nil} {
		t.Run(name, func(t *testing.T) {
			client := planetv1connect.NewAdminServiceClient(http.DefaultClient, url, options...)

			ban, err := client.BanPlayer(t.Context(), connect.NewRequest(&planetv1.BanPlayerRequest{
				Scope: "203.0.113.7", Duration: durationpb.New(time.Hour),
			}))
			require.NoError(t, err)
			assert.Equal(t, ban_player_usecase.In{Scope: "203.0.113.7", Duration: time.Hour}, banned.asked)
			assert.Equal(t, "203.0.113.7", ban.Msg.GetScope())
			assert.Equal(t, uint32(2), ban.Msg.GetOffence())
			assert.Equal(t, until, ban.Msg.GetBannedUntil().AsTime())
			assert.True(t, ban.Msg.GetEnforced())

			unban, err := client.UnbanPlayer(t.Context(), connect.NewRequest(&planetv1.UnbanPlayerRequest{Scope: "203.0.113.7"}))
			require.NoError(t, err)
			assert.Equal(t, uint32(2), unban.Msg.GetOffence())

			inspected, err := client.InspectPlayer(t.Context(), connect.NewRequest(&planetv1.InspectPlayerRequest{
				AccountId: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10",
			}))
			require.NoError(t, err)
			assert.Equal(t, "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", inspected.Msg.GetAccountId())
			assert.Equal(t, uint32(42), inspected.Msg.GetClicks())
			require.Len(t, inspected.Msg.GetReadings(), 1)
			assert.Equal(t, "suspect", inspected.Msg.GetReadings()[0].GetLevel())

			_, err = client.TopPlayers(t.Context(), connect.NewRequest(&planetv1.TopPlayersRequest{}))
			assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err), "planet's other calls are still planet's")
		})
	}
}
