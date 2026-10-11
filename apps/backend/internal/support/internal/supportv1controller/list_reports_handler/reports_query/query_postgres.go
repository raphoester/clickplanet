package reports_query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Authors interface {
	Authors(ctx context.Context, accounts []cpsession.AccountID) (map[cpsession.AccountID]*playerv1.Author, error)
}

const (
	DefaultLimit = 100
	MaxLimit     = 500
)

var ErrUnknownReason = errors.New("a stored report has a reason the wire cannot carry")

var reasons = map[string]supportv1.ReportReason{
	"cheating":   supportv1.ReportReason_REPORT_REASON_CHEATING,
	"chat_abuse": supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE,
	"bad_name":   supportv1.ReportReason_REPORT_REASON_BAD_NAME,
	"other":      supportv1.ReportReason_REPORT_REASON_OTHER,
}

func NewPostgresQuery(db cppg.Querier, authors Authors) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors}
}

type PostgresQuery struct {
	db      cppg.Querier
	authors Authors
}

const reports = `
	SELECT id, reporter_account_id, reported_account_id, reason, text, message_id, reported_at
	FROM reports
	WHERE $1::timestamptz IS NULL OR reported_at >= $1
	ORDER BY reported_at DESC, id DESC
	LIMIT $2
`

type row struct {
	report             *supportv1.Report
	reporter, reported cpsession.AccountID
}

// A zero since is every report kept; a zero limit is DefaultLimit, and none passes MaxLimit.
func (q *PostgresQuery) Reports(ctx context.Context, since time.Time, limit uint32) (*supportv1.ListReportsResponse, error) {
	rows, err := q.rows(ctx, since, limit)
	if err != nil {
		return nil, err
	}

	named, err := q.authors.Authors(ctx, everyone(rows))
	if err != nil {
		return nil, fmt.Errorf("failed to read who the reports name: %w", err)
	}

	answer := &supportv1.ListReportsResponse{Reports: make([]*supportv1.Report, 0, len(rows))}
	for _, row := range rows {
		row.report.ReporterName = named[row.reporter].GetName()
		row.report.ReportedName = named[row.reported].GetName()
		answer.Reports = append(answer.Reports, row.report)
	}
	return answer, nil
}

func (q *PostgresQuery) rows(ctx context.Context, since time.Time, limit uint32) ([]row, error) {
	found, err := q.db.QueryContext(ctx, reports, sql.NullTime{Time: since.UTC(), Valid: !since.IsZero()}, limitOf(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to read the reports: %w", err)
	}
	defer func() { _ = found.Close() }()

	var read []row
	for found.Next() {
		var (
			report             = &supportv1.Report{}
			reporter, reported uuid.UUID
			reason             string
			at                 time.Time
		)
		if err := found.Scan(&report.Id, &reporter, &reported, &reason, &report.Text, &report.MessageId, &at); err != nil {
			return nil, fmt.Errorf("failed to read a report: %w", err)
		}

		known, ok := reasons[reason]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownReason, reason)
		}
		report.Reason = known
		report.ReporterAccountId, report.ReportedAccountId = reporter.String(), reported.String()
		report.ReportedAt = timestamppb.New(at)
		read = append(read, row{report: report, reporter: cpsession.AccountID(reporter), reported: cpsession.AccountID(reported)})
	}
	if err := found.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the reports: %w", err)
	}
	return read, nil
}

func everyone(rows []row) []cpsession.AccountID {
	seen := map[cpsession.AccountID]int{}
	var accounts []cpsession.AccountID
	for _, row := range rows {
		for _, account := range []cpsession.AccountID{row.reporter, row.reported} {
			if seen[account]++; seen[account] == 1 {
				accounts = append(accounts, account)
			}
		}
	}
	return accounts
}

func limitOf(asked uint32) int {
	switch {
	case asked == 0:
		return DefaultLimit
	case asked > MaxLimit:
		return MaxLimit
	default:
		return int(asked)
	}
}
