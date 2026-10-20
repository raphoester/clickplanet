package place_reports_query

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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
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

var (
	ErrUnknownReason  = errors.New("a stored place report has a reason the wire cannot carry")
	ErrUnknownOutcome = errors.New("a stored decision has an outcome the wire cannot carry")
)

var reasons = map[string]supportv1.PlaceReportReason{
	"hate":   supportv1.PlaceReportReason_PLACE_REPORT_REASON_HATE,
	"sexual": supportv1.PlaceReportReason_PLACE_REPORT_REASON_SEXUAL,
	"other":  supportv1.PlaceReportReason_PLACE_REPORT_REASON_OTHER,
}

var outcomes = map[string]supportv1.ReportOutcome{
	"acted":     supportv1.ReportOutcome_REPORT_OUTCOME_ACTED,
	"dismissed": supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED,
}

func NewPostgresQuery(db cppg.Querier, authors Authors) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors}
}

type PostgresQuery struct {
	db      cppg.Querier
	authors Authors
}

const placeReports = `
	SELECT place_reports.id, reporter_account_id, tile_id, reason, text, created_at,
		place_report_decisions.outcome, place_report_decisions.comment, place_report_decisions.decided_at
	FROM place_reports
	LEFT JOIN place_report_decisions ON place_report_decisions.report_id = place_reports.id
	WHERE ($1::timestamptz IS NULL OR created_at >= $1) AND (NOT $3 OR place_report_decisions.report_id IS NULL)
	ORDER BY created_at DESC, place_reports.id DESC
	LIMIT $2
`

type row struct {
	report   *supportv1.PlaceReport
	reporter cpsession.AccountID
}

// A zero since is every report kept; a zero limit is DefaultLimit, and none passes MaxLimit.
func (q *PostgresQuery) PlaceReports(
	ctx context.Context, since time.Time, limit uint32, openOnly bool,
) (*supportv1.ListPlaceReportsResponse, error) {
	rows, err := q.rows(ctx, since, limit, openOnly)
	if err != nil {
		return nil, err
	}

	named, err := q.authors.Authors(ctx, reporters(rows))
	if err != nil {
		return nil, fmt.Errorf("failed to read who sent the reports: %w", err)
	}

	answer := &supportv1.ListPlaceReportsResponse{Reports: make([]*supportv1.PlaceReport, 0, len(rows))}
	for _, row := range rows {
		row.report.ReporterName = named[row.reporter].GetName()
		answer.Reports = append(answer.Reports, row.report)
	}
	return answer, nil
}

func (q *PostgresQuery) rows(ctx context.Context, since time.Time, limit uint32, openOnly bool) ([]row, error) {
	found, err := q.db.QueryContext(ctx, placeReports,
		sql.NullTime{Time: since.UTC(), Valid: !since.IsZero()}, limitOf(limit), openOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to read the place reports: %w", err)
	}
	defer func() { _ = found.Close() }()

	var read []row
	for found.Next() {
		var (
			report           = &supportv1.PlaceReport{}
			reporter         uuid.NullUUID
			tile             int64
			reason           string
			at               time.Time
			outcome, comment sql.NullString
			decidedAt        sql.NullTime
		)
		if err := found.Scan(&report.Id, &reporter, &tile, &reason, &report.Text, &at,
			&outcome, &comment, &decidedAt); err != nil {
			return nil, fmt.Errorf("failed to read a place report: %w", err)
		}

		known, ok := reasons[reason]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownReason, reason)
		}
		report.Reason = known
		report.TileId = uint32(tile) //nolint:gosec // a tile id, kept from a uint32.
		report.CreatedAt = timestamppb.New(at)
		if outcome.Valid {
			decided, ok := outcomes[outcome.String]
			if !ok {
				return nil, fmt.Errorf("%w: %q", ErrUnknownOutcome, outcome.String)
			}
			report.Decision = &supportv1.ReportDecision{
				Outcome: decided, Comment: comment.String, DecidedAt: timestamppb.New(decidedAt.Time),
			}
		}

		each := row{report: report}
		if reporter.Valid {
			report.ReporterAccountId = reporter.UUID.String()
			each.reporter = cpsession.AccountID(reporter.UUID)
		}
		read = append(read, each)
	}
	if err := found.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the place reports: %w", err)
	}
	return read, nil
}

func reporters(rows []row) []cpsession.AccountID {
	seen := cpcolls.NewSet[cpsession.AccountID]()
	var accounts []cpsession.AccountID
	for _, row := range rows {
		if row.reporter == cpsession.NoAccount || seen.Contains(row.reporter) {
			continue
		}
		seen.Add(row.reporter)
		accounts = append(accounts, row.reporter)
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
