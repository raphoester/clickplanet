package ssh_stats_store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/usecases/take_snapshot_usecase"
)

const OpenSSH = "ssh"

// The body comes on the standard input, never the command line: a part is far longer than a command may be, and holds
// what Zernio wrote, which no shell should read. The method is one of ours, never an input.
const operatorTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
	" --header 'Content-Type: application/json' --post-file=/dev/stdin" +
	" http://127.0.0.1:8081/clips.v1.AdminService/"

const (
	keepSnapshot  = "KeepSnapshot"
	listSnapshots = "ListSnapshots"
	listRows      = "ListRows"
)

// A part is a call: about 3 KB a row, so under a megabyte.
const rowsAPart = 200

// The most ListRows answers a page.
const rowsAPage = 2000

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

// host: user@host of the machine the backend runs on. client: the ssh program.
func New(host, client string) (*Store, error) {
	if host == "" {
		return nil, errors.New("the ssh host is empty")
	}
	return &Store{host: host, client: client}, nil
}

// The backend's operator tool, as fetch asks it for a replay: it keeps the snapshot in postgres.
type Store struct {
	host   string
	client string
}

var _ take_snapshot_usecase.Store = (*Store)(nil)

// In parts, each with the report: a part kept twice is kept once, so a run made again keeps one snapshot.
func (s *Store) Keep(ctx context.Context, report stats.Report, rows []stats.Row) error {
	summary := summaryOf(report)
	for start := 0; start == 0 || start < len(rows); start += rowsAPart {
		part := rows[start:min(start+rowsAPart, len(rows))]
		if err := s.keep(ctx, summary, part); err != nil {
			return fmt.Errorf("failed to keep rows %d to %d of %d: %w", start+1, start+len(part), len(rows), err)
		}
	}
	return nil
}

func (s *Store) keep(ctx context.Context, summary *clipsv1.StatsSnapshot, rows []stats.Row) error {
	request := &clipsv1.KeepSnapshotRequest{Snapshot: summary}
	for _, row := range rows {
		document, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("failed to write the row of %s on %s: %w", row.PostID, row.Platform, err)
		}
		request.Rows = append(request.Rows, &clipsv1.StatsRow{
			TakenAt: timestamppb.New(row.TakenAt), PostId: string(row.PostID), Platform: row.Platform, Document: string(document),
		})
	}
	body, err := protojson.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to write the request: %w", err)
	}

	answered, err := s.ask(ctx, keepSnapshot, body)
	if err != nil {
		return err
	}
	var kept clipsv1.KeepSnapshotResponse
	if err := decoder.Unmarshal(answered, &kept); err != nil {
		return fmt.Errorf("the backend answered what is not a clips.v1.KeepSnapshotResponse: %w", err)
	}
	if int(kept.GetRows()) != len(rows) {
		return fmt.Errorf("the backend kept %d rows of %d", kept.GetRows(), len(rows))
	}
	return nil
}

func summaryOf(report stats.Report) *clipsv1.StatsSnapshot {
	summary := &clipsv1.StatsSnapshot{
		TakenAt: timestamppb.New(report.TakenAt), Since: timestamppb.New(report.Since),
		Posts: count(report.Posts), NotOut: count(report.NotOut), Rows: count(report.RowCount()),
		Syncing: count(report.Syncing), Curves: count(report.Curves),
	}
	for _, lane := range report.TikTokAccounts {
		summary.TiktokAccounts = append(summary.TiktokAccounts, &clipsv1.TikTokAccount{
			AccountId: lane.ID, Username: lane.Username, AccountType: lane.Type,
			CommercialMusic: lane.CommercialMusic, CommercialMusicRefusal: lane.CommercialMusicRefusal,
			Rows: count(lane.Rows), BusinessRows: count(lane.BusinessRows), Live: lane.Live(),
		})
	}
	for _, unread := range report.Unread {
		summary.Unread = append(summary.Unread, &clipsv1.UnreadPost{PostId: string(unread.PostID), Reason: unread.Reason})
	}
	return summary
}

func count(n int) uint32 {
	return uint32(max(n, 0)) //nolint:gosec // a count of posts and rows.
}

// The rows of the newest snapshot kept, a page at a time; none before the first.
func (s *Store) Last(ctx context.Context) ([]stats.Row, error) {
	var snapshots clipsv1.ListSnapshotsResponse
	if err := s.call(ctx, listSnapshots, &clipsv1.ListSnapshotsRequest{Limit: 1}, &snapshots); err != nil {
		return nil, err
	}
	if len(snapshots.GetSnapshots()) == 0 {
		return nil, nil
	}
	takenAt := snapshots.GetSnapshots()[0].GetTakenAt()

	var rows []stats.Row
	request := &clipsv1.ListRowsRequest{
		Since: takenAt, Until: timestamppb.New(takenAt.AsTime().Add(time.Second)), Limit: rowsAPage,
	}
	for {
		var page clipsv1.ListRowsResponse
		if err := s.call(ctx, listRows, request, &page); err != nil {
			return nil, err
		}
		for _, kept := range page.GetRows() {
			var row stats.Row
			if err := json.Unmarshal([]byte(kept.GetDocument()), &row); err != nil {
				return nil, fmt.Errorf("the row of %s on %s is not a row: %w", kept.GetPostId(), kept.GetPlatform(), err)
			}
			rows = append(rows, row)
		}
		if page.GetNextPage() == "" {
			return rows, nil
		}
		request.Page = page.GetNextPage()
	}
}

func (s *Store) call(ctx context.Context, method string, request, response proto.Message) error {
	body, err := protojson.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to write the request of %s: %w", method, err)
	}
	answered, err := s.ask(ctx, method, body)
	if err != nil {
		return err
	}
	if err := decoder.Unmarshal(answered, response); err != nil {
		return fmt.Errorf("the backend answered %s with what is not its response: %w", method, err)
	}
	return nil
}

func (s *Store) ask(ctx context.Context, method string, body []byte) ([]byte, error) {
	var answered, complained bytes.Buffer
	// "--" ends the options: a host that starts with a dash is a host, never an option of ssh.
	call := exec.CommandContext(ctx, s.client, "--", s.host, operatorTool+method) //nolint:gosec // the operator of the job names the host, and the method is a constant.
	call.Stdin = bytes.NewReader(body)
	call.Stdout = &answered
	call.Stderr = &complained
	if err := call.Run(); err != nil {
		return nil, fmt.Errorf("ssh failed: %w: %s", err, strings.TrimSpace(complained.String()))
	}
	return answered.Bytes(), nil
}
