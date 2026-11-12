package clips_test

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1/clipsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis/cpredistest"
)

func listen(t *testing.T) net.Listener {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return listener
}

func TestTheModuleServesItsOperatorAndThePlayersOnceTheServerRuns(t *testing.T) {
	config := clips.ModuleConfig{
		Database: cppgtest.StartServer(t).ConfigFor("clips"), Redis: cpredistest.StartServer(t).Config(),
		Judge: clips.NameJudgeConfig{Anthropic: cpanthropic.Config{APIKey: "a-test-key", BaseURL: "http://127.0.0.1:1"}},
	}
	public, console, internal := listen(t), listen(t), listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server: cpbootstrap.ServerConfig{
				BindAddress: public.Addr().String(), ConsoleBindAddress: console.Addr().String(),
				InternalBindAddress: internal.Addr().String(),
			},
			Logger:  slog.New(slog.DiscardHandler),
			Modules: []cpbootstrap.Module{clips.NewModule(config)},
		}, public, console, internal)
	}()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	client := clipsv1connect.NewAdminServiceClient(http.DefaultClient, "http://"+console.Addr().String())
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := client.ListNameVerdicts(t.Context(), connect.NewRequest(&clipsv1.ListNameVerdictsRequest{}))
		assert.NoError(c, err)
	}, time.Minute, 100*time.Millisecond, "the module serves once the server runs, its pool open")

	players := clipsv1connect.NewClipsServiceClient(http.DefaultClient, "http://"+public.Addr().String())
	_, err := players.GetAppearances(t.Context(), connect.NewRequest(&clipsv1.GetAppearancesRequest{}))
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "the posts that name a player are its own")
	_, err = players.GetShownNames(t.Context(), connect.NewRequest(&clipsv1.GetShownNamesRequest{}))
	assert.NoError(t, err, "a guest reads the names a video may show")
}

func TestAConfigWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, clips.ModuleConfig{}.Validate(), "clips.database")
}

func TestAConfigWithNoRedisIsRefused(t *testing.T) {
	database := cppg.Config{Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres", SSLMode: "disable", Schema: "clips"}

	assert.ErrorContains(t, clips.ModuleConfig{Database: database}.Validate(), "clips.redis")
}
