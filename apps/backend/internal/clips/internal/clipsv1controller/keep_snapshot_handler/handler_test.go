package keep_snapshot_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots"
)

type useCase struct {
	kept []snapshots.PartSnapshot
}

func (u *useCase) Execute(_ context.Context, part snapshots.Part) (int, error) {
	u.kept = append(u.kept, part.TakeSnapshot())
	return len(part.TakeSnapshot().Rows), nil
}

var takenAt = time.Date(2026, 10, 25, 5, 41, 7, 0, time.UTC)

func keep(t *testing.T, msg *clipsv1.KeepSnapshotRequest) (*useCase, *clipsv1.KeepSnapshotResponse, error) {
	t.Helper()

	kept := &useCase{}
	answer, err := keep_snapshot_handler.New(kept).KeepSnapshot(t.Context(), connect.NewRequest(msg))
	if err != nil {
		return kept, nil, err //nolint:wrapcheck // the test reads the handler's own error.
	}
	return kept, answer.Msg, nil
}

func TestAPartIsKeptWithItsSummaryAsProtojsonWritesIt(t *testing.T) {
	kept, answer, err := keep(t, &clipsv1.KeepSnapshotRequest{
		Snapshot: &clipsv1.StatsSnapshot{TakenAt: timestamppb.New(takenAt), Posts: 2},
		Rows: []*clipsv1.StatsRow{
			{TakenAt: timestamppb.New(takenAt), PostId: "post-1", Platform: "youtube", Document: `{"version":1}`},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, uint32(1), answer.GetRows())
	require.Len(t, kept.kept, 1)
	assert.Equal(t, takenAt, kept.kept[0].TakenAt)
	assert.JSONEq(t, `{"takenAt":"2026-10-25T05:41:07Z","posts":2}`, string(kept.kept[0].Summary))
	assert.Equal(t, "youtube", kept.kept[0].Rows[0].Platform)
}

func TestAPartThatIsNotWholeIsTheCallersFault(t *testing.T) {
	for name, msg := range map[string]*clipsv1.KeepSnapshotRequest{
		"no snapshot": {},
		"a row of another snapshot": {
			Snapshot: &clipsv1.StatsSnapshot{TakenAt: timestamppb.New(takenAt)},
			Rows:     []*clipsv1.StatsRow{{TakenAt: timestamppb.New(takenAt.Add(time.Hour)), PostId: "p", Platform: "tiktok", Document: `{}`}},
		},
		"a row with no time": {
			Snapshot: &clipsv1.StatsSnapshot{TakenAt: timestamppb.New(takenAt)},
			Rows:     []*clipsv1.StatsRow{{PostId: "p", Platform: "tiktok", Document: `{}`}},
		},
		"a document that is no object": {
			Snapshot: &clipsv1.StatsSnapshot{TakenAt: timestamppb.New(takenAt)},
			Rows:     []*clipsv1.StatsRow{{TakenAt: timestamppb.New(takenAt), PostId: "p", Platform: "tiktok", Document: `[1]`}},
		},
	} {
		kept, _, err := keep(t, msg)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), name)
		assert.Empty(t, kept.kept, name)
	}
}
