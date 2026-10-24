package keep_snapshot_handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots"
)

type UseCase interface {
	Execute(ctx context.Context, part snapshots.Part) (int, error)
}

func New(useCase UseCase) KeepSnapshotHandler {
	return KeepSnapshotHandler{useCase: useCase}
}

type KeepSnapshotHandler struct {
	useCase UseCase
}

func (h KeepSnapshotHandler) KeepSnapshot(
	ctx context.Context,
	req *connect.Request[clipsv1.KeepSnapshotRequest],
) (*connect.Response[clipsv1.KeepSnapshotResponse], error) {
	part, err := partOf(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	kept, err := h.useCase.Execute(ctx, part)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&clipsv1.KeepSnapshotResponse{Rows: uint32(kept)}), nil //nolint:gosec // the rows of one call.
}

func partOf(msg *clipsv1.KeepSnapshotRequest) (snapshots.Part, error) {
	summary := msg.GetSnapshot()
	if summary.GetTakenAt() == nil {
		return snapshots.Part{}, snapshots.ErrNoSnapshot
	}
	written, err := protojson.Marshal(summary)
	if err != nil {
		return snapshots.Part{}, fmt.Errorf("failed to write the summary: %w", err)
	}

	rows := make([]snapshots.Row, 0, len(msg.GetRows()))
	times := make([]time.Time, 0, len(msg.GetRows()))
	for _, row := range msg.GetRows() {
		if row.GetTakenAt() == nil {
			return snapshots.Part{}, errors.New("a row with no time")
		}
		rows = append(rows, snapshots.Row{
			PostID: row.GetPostId(), Platform: row.GetPlatform(), Document: json.RawMessage(row.GetDocument()),
		})
		times = append(times, row.GetTakenAt().AsTime())
	}
	return snapshots.NewPart(summary.GetTakenAt().AsTime(), written, rows, times) //nolint:wrapcheck // the domain named what is wrong.
}
