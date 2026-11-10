// The snapshots, in publishing's schema: a summary a snapshot, and a row a platform of a post.
package postgres_snapshot_store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/stats/usecases/take_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ take_snapshot_usecase.Store = (*Store)(nil)

// In one transaction: the summary and every row, or none of them. A snapshot kept again replaces the one kept, row by
// row, so a run made again keeps one snapshot.
func (s *Store) Keep(ctx context.Context, report stats.Report, rows []stats.Row) error {
	summary, err := protojson.Marshal(SummaryOf(report))
	if err != nil {
		return fmt.Errorf("failed to write the summary of a snapshot: %w", err)
	}
	return s.db.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the transaction's own errors say what failed.
		tx := s.db.On(ctx)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO snapshots (taken_at, summary) VALUES ($1, $2)
			ON CONFLICT (taken_at) DO UPDATE SET summary = EXCLUDED.summary
		`, report.TakenAt.UTC(), string(summary)); err != nil {
			return fmt.Errorf("failed to keep the summary of a snapshot: %w", err)
		}
		for _, row := range rows {
			document, err := json.Marshal(row)
			if err != nil {
				return fmt.Errorf("failed to write the row of %s on %s: %w", row.PostID, row.Platform, err)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO snapshot_rows (taken_at, post_id, platform, document) VALUES ($1, $2, $3, $4)
				ON CONFLICT (taken_at, post_id, platform) DO UPDATE SET document = EXCLUDED.document
			`, row.TakenAt.UTC(), string(row.PostID), row.Platform, string(document)); err != nil {
				return fmt.Errorf("failed to keep the row of %s on %s: %w", row.PostID, row.Platform, err)
			}
		}
		return nil
	})
}

// When the newest snapshot was taken: zero for none.
func (s *Store) Latest(ctx context.Context) (time.Time, error) {
	var latest sql.NullTime
	if err := s.db.On(ctx).QueryRowContext(ctx, `SELECT max(taken_at) FROM snapshots`).Scan(&latest); err != nil {
		return time.Time{}, fmt.Errorf("failed to read the newest snapshot: %w", err)
	}
	return latest.Time.UTC(), nil
}

// What a snapshot read and kept, as ListSnapshots answers it: counts and states, never a metric.
func SummaryOf(report stats.Report) *publishingv1.StatsSnapshot {
	summary := &publishingv1.StatsSnapshot{
		TakenAt: timestamppb.New(report.TakenAt), Since: timestamppb.New(report.Since),
		Posts: count(report.Posts), NotOut: count(report.NotOut), Rows: count(report.RowCount()),
		Syncing: count(report.Syncing), Curves: count(report.Curves),
	}
	for _, lane := range report.TikTokAccounts {
		summary.TiktokAccounts = append(summary.TiktokAccounts, &publishingv1.TikTokAccount{
			AccountId: lane.ID, Username: lane.Username, AccountType: lane.Type,
			CommercialMusic: lane.CommercialMusic, CommercialMusicRefusal: lane.CommercialMusicRefusal,
			Rows: count(lane.Rows), BusinessRows: count(lane.BusinessRows), Live: lane.Live(),
		})
	}
	for _, unread := range report.Unread {
		summary.Unread = append(summary.Unread, &publishingv1.UnreadPost{PostId: string(unread.PostID), Reason: unread.Reason})
	}
	return summary
}

func count(n int) uint32 {
	return uint32(max(n, 0)) //nolint:gosec // a count of posts and rows.
}
