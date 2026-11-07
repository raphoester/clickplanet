package player_reported_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/usecases/judge_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/player_reported_subscriber"
)

var (
	at       = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)
	raider   = messages.AccountID{15: 1}
	reporter = messages.AccountID{15: 2}
	report   = judgements.ReportID{15: 9}
)

type recording struct{ reports []judgements.Report }

func (r *recording) Execute(_ context.Context, report judgements.Report) (judge_report_usecase.Out, error) {
	r.reports = append(r.reports, report)
	return judge_report_usecase.Out{}, nil
}

func TestAReportIsHandedToTheJudgeInTheChatsWords(t *testing.T) {
	useCase := &recording{}

	err := player_reported_subscriber.New(useCase).Handle(t.Context(), &supportv1.PlayerReported{
		ReportId: report.String(), ReporterAccountId: reporter.String(), ReportedAccountId: raider.String(),
		Reason: supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE, Text: "slurs", MessageId: "m-2",
		ReportedAt: timestamppb.New(at),
	})

	require.NoError(t, err)
	assert.Equal(t, []judgements.Report{{
		ID: report, Reporter: reporter, Reported: raider, Reason: judgements.ReasonChatAbuse, Text: "slurs", Message: "m-2",
		At: at,
	}}, useCase.reports)
}

func TestEveryReasonIsTheChatsOwn(t *testing.T) {
	for wire, reason := range map[supportv1.ReportReason]judgements.Reason{
		supportv1.ReportReason_REPORT_REASON_CHEATING:    judgements.ReasonCheating,
		supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE:  judgements.ReasonChatAbuse,
		supportv1.ReportReason_REPORT_REASON_BAD_NAME:    judgements.ReasonBadName,
		supportv1.ReportReason_REPORT_REASON_OTHER:       judgements.ReasonOther,
		supportv1.ReportReason_REPORT_REASON_UNSPECIFIED: judgements.ReasonOther,
	} {
		useCase := &recording{}
		require.NoError(t, player_reported_subscriber.New(useCase).Handle(t.Context(), &supportv1.PlayerReported{
			ReportId: report.String(), ReportedAccountId: raider.String(), Reason: wire,
		}))
		assert.Equal(t, reason, useCase.reports[0].Reason, wire)
	}
}

func TestAReportWithNoIDOrNoPlayerIsRefused(t *testing.T) {
	useCase := &recording{}

	require.ErrorIs(t, player_reported_subscriber.New(useCase).Handle(t.Context(),
		&supportv1.PlayerReported{ReportId: "nope", ReportedAccountId: raider.String()}), judgements.ErrInvalidReport)
	require.ErrorIs(t, player_reported_subscriber.New(useCase).Handle(t.Context(),
		&supportv1.PlayerReported{ReportId: report.String()}), judgements.ErrInvalidReport)
	assert.Empty(t, useCase.reports)
}
