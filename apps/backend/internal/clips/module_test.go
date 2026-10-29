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
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1/clipsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

func listen(t *testing.T) net.Listener {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return listener
}

func TestTheModuleKeepsAndReadsSnapshotsOnTheConsoleListenerOnceTheServerRuns(t *testing.T) {
	config := clips.ModuleConfig{Database: cppg.StartTestServer(t).ConfigFor("clips"), Redis: cpredis.StartTestServer(t).Config()}
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
	takenAt := timestamppb.New(time.Date(2026, 10, 25, 5, 41, 7, 0, time.UTC))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := client.ListSnapshots(t.Context(), connect.NewRequest(&clipsv1.ListSnapshotsRequest{}))
		assert.NoError(c, err)
	}, time.Minute, 100*time.Millisecond, "the module serves once the server runs, its pool open")

	kept, err := client.KeepSnapshot(t.Context(), connect.NewRequest(&clipsv1.KeepSnapshotRequest{
		Snapshot: &clipsv1.StatsSnapshot{TakenAt: takenAt, Posts: 1, Rows: 1},
		Rows:     []*clipsv1.StatsRow{{TakenAt: takenAt, PostId: "post-1", Platform: "tiktok", Document: `{"version":1}`}},
	}))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), kept.Msg.GetRows())

	rows, err := client.ListRows(t.Context(), connect.NewRequest(&clipsv1.ListRowsRequest{}))
	require.NoError(t, err)
	require.Len(t, rows.Msg.GetRows(), 1)
	assert.JSONEq(t, `{"version":1}`, rows.Msg.GetRows()[0].GetDocument())

	_, err = clipsv1connect.NewClipsServiceClient(http.DefaultClient, "http://"+public.Addr().String()).
		GetAppearances(t.Context(), connect.NewRequest(&clipsv1.GetAppearancesRequest{}))
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "the posts that name a player are its own")
}

func TestAConfigWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, clips.ModuleConfig{}.Validate(), "clips.database")
}

func TestAConfigWithNoRedisIsRefused(t *testing.T) {
	database := cppg.Config{Host: "localhost", Port: "5432", User: "postgres", DBName: "postgres", SSLMode: "disable", Schema: "clips"}

	assert.ErrorContains(t, clips.ModuleConfig{Database: database}.Validate(), "clips.redis")
}
