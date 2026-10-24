package keep_snapshot_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipstatsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clipstats/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/clipstatsv1controller/keep_snapshot_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clipstats/internal/snapshots"
)

type useCase struct {
	kept []snapshots.PartSnapshot
}

func (u *useCase) Execute(_ context.Context, part snapshots.Part) (int, error) {
	u.kept = append(u.kept, part.TakeSnapshot())
	return len(part.TakeSnapshot().Rows), nil
}

var takenAt = time.Date(2026, 10, 25, 5, 41, 7, 0, time.UTC)

func keep(t *testing.T, msg *clipstatsv1.KeepSnapshotRequest) (*useCase, *clipstatsv1.KeepSnapshotResponse, error) {
	t.Helper()

	kept := &useCase{}
	answer, err := keep_snapshot_handler.New(kept).KeepSnapshot(t.Context(), connect.NewRequest(msg))
	if err != nil {
		return kept, nil, err //nolint:wrapcheck // the test reads the handler's own error.
	}
	return kept, answer.Msg, nil
}

func TestAPartIsKeptWithItsSummaryAsProtojsonWritesIt(t *testing.T) {
	kept, answer, err := keep(t, &clipstatsv1.KeepSnapshotRequest{
		Snapshot: &clipstatsv1.Snapshot{TakenAt: timestamppb.New(takenAt), Posts: 2},
		Rows: []*clipstatsv1.Row{
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
	for name, msg := range map[string]*clipstatsv1.KeepSnapshotRequest{
		"no snapshot": {},
		"a row of another snapshot": {
			Snapshot: &clipstatsv1.Snapshot{TakenAt: timestamppb.New(takenAt)},
			Rows:     []*clipstatsv1.Row{{TakenAt: timestamppb.New(takenAt.Add(time.Hour)), PostId: "p", Platform: "tiktok", Document: `{}`}},
		},
		"a row with no time": {
			Snapshot: &clipstatsv1.Snapshot{TakenAt: timestamppb.New(takenAt)},
			Rows:     []*clipstatsv1.Row{{PostId: "p", Platform: "tiktok", Document: `{}`}},
		},
		"a document that is no object": {
			Snapshot: &clipstatsv1.Snapshot{TakenAt: timestamppb.New(takenAt)},
			Rows:     []*clipstatsv1.Row{{TakenAt: timestamppb.New(takenAt), PostId: "p", Platform: "tiktok", Document: `[1]`}},
		},
	} {
		kept, _, err := keep(t, msg)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), name)
		assert.Empty(t, kept.kept, name)
	}
}
