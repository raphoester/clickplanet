package report_player_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type Reports interface {
	FiledBy(ctx context.Context, reporter reports.AccountID, about reports.AccountID) ([]reports.Report, error)
	Save(ctx context.Context, reports ...reports.Report) error
}

type In struct {
	Reporter  reports.AccountID
	Reported  reports.AccountID
	Complaint reports.Complaint
}

const timeout = 5 * time.Second

func New(
	ids reports.IDProvider, clock cptime.Clock, db cpatomic.Transactioner, kept Reports, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{ids: ids, clock: clock, db: db, reports: kept, outbox: outbox}
}

type UseCase struct {
	ids     reports.IDProvider
	clock   cptime.Clock
	db      cpatomic.Transactioner
	reports Reports
	outbox  cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	earlier, err := u.reports.FiledBy(ctx, in.Reporter, in.Reported)
	if err != nil {
		return fmt.Errorf("failed to read what the reporter already filed about the player: %w", err)
	}

	id, err := u.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to draw a report id: %w", err)
	}

	report, err := reports.NewReport(id, in.Reporter, in.Reported, in.Complaint, u.clock.Now(), earlier)
	if err != nil {
		return fmt.Errorf("cannot take the report: %w", err)
	}

	// Never a report the chat is not told of, nor an event of a report not kept.
	err = u.db.InTx(ctx, func(ctx context.Context) error {
		if err := u.reports.Save(ctx, report); err != nil {
			return fmt.Errorf("failed to keep the report: %w", err)
		}
		if err := u.outbox.Record(ctx, reportedOf(report)); err != nil {
			return fmt.Errorf("failed to record that the player was reported: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to file the report: %w", err)
	}
	return nil
}

func reportedOf(report reports.Report) *supportv1.PlayerReported {
	kept := report.TakeSnapshot()
	return &supportv1.PlayerReported{
		ReportId:          uuid.UUID(kept.ID).String(),
		ReporterAccountId: kept.Reporter.String(),
		ReportedAccountId: kept.Reported.String(),
		Reason:            reasons[kept.Complaint.Reason],
		Text:              kept.Complaint.Text,
		MessageId:         string(kept.Complaint.Message),
		ReportedAt:        timestamppb.New(kept.At),
	}
}

var reasons = map[reports.Reason]supportv1.ReportReason{
	reports.ReasonCheating:  supportv1.ReportReason_REPORT_REASON_CHEATING,
	reports.ReasonChatAbuse: supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE,
	reports.ReasonBadName:   supportv1.ReportReason_REPORT_REASON_BAD_NAME,
	reports.ReasonOther:     supportv1.ReportReason_REPORT_REASON_OTHER,
}
