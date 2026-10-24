package clipstats_test

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

	clipstatsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clipstats/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/clipstats/v1/clipstatsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func listen(t *testing.T) net.Listener {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return listener
}

func TestTheModuleKeepsAndReadsSnapshotsOnTheConsoleListenerOnceTheServerRuns(t *testing.T) {
	database := cppg.StartTestServer(t).ConfigFor("clipstats")
	public, console := listen(t), listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server:  cpbootstrap.ServerConfig{BindAddress: public.Addr().String(), ConsoleBindAddress: console.Addr().String()},
			Logger:  slog.New(slog.DiscardHandler),
			Modules: []cpbootstrap.Module{clipstats.NewModule(clipstats.Config{Database: database})},
		}, public, console)
	}()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	client := clipstatsv1connect.NewAdminServiceClient(http.DefaultClient, "http://"+console.Addr().String())
	takenAt := timestamppb.New(time.Date(2026, 10, 25, 5, 41, 7, 0, time.UTC))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := client.ListSnapshots(t.Context(), connect.NewRequest(&clipstatsv1.ListSnapshotsRequest{}))
		assert.NoError(c, err)
	}, time.Minute, 100*time.Millisecond, "the module serves once the server runs, its pool open")

	kept, err := client.KeepSnapshot(t.Context(), connect.NewRequest(&clipstatsv1.KeepSnapshotRequest{
		Snapshot: &clipstatsv1.Snapshot{TakenAt: takenAt, Posts: 1, Rows: 1},
		Rows:     []*clipstatsv1.Row{{TakenAt: takenAt, PostId: "post-1", Platform: "tiktok", Document: `{"version":1}`}},
	}))
	require.NoError(t, err)
	assert.Equal(t, uint32(1), kept.Msg.GetRows())

	rows, err := client.ListRows(t.Context(), connect.NewRequest(&clipstatsv1.ListRowsRequest{}))
	require.NoError(t, err)
	require.Len(t, rows.Msg.GetRows(), 1)
	assert.JSONEq(t, `{"version":1}`, rows.Msg.GetRows()[0].GetDocument())
}

func TestAConfigWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, clipstats.Config{}.Validate(), "clipstats.database")
}
