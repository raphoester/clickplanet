package player_reported_subscriber

import (
	"context"
	"time"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/usecases/judge_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// A judgement asks the model, which answers in a minute at most, and tries once more.
const timeout = 3 * time.Minute

func New(useCase judge_report_usecase.Executor) Subscriber {
	return Subscriber{useCase: useCase}
}

// support.v1.PlayerReported, in the chat's words: a report for the judge.
type Subscriber struct {
	useCase judge_report_usecase.Executor
}

var _ cpredis.Handler[*supportv1.PlayerReported] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *supportv1.PlayerReported) error {
	id, err := judgements.ReportIDOf(event.GetReportId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel says what is wrong.
	}
	reported := messages.AccountIDOf(event.GetReportedAccountId())
	if reported == messages.NoAccount {
		return judgements.ErrInvalidReport
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err = s.useCase.Execute(ctx, judgements.Report{
		ID:       id,
		Reporter: messages.AccountIDOf(event.GetReporterAccountId()),
		Reported: reported,
		Reason:   reasonOf(event.GetReason()),
		Text:     event.GetText(),
		Message:  messages.MessageID(event.GetMessageId()),
		At:       event.GetReportedAt().AsTime(),
	})
	return err //nolint:wrapcheck // the use case named it.
}

// A reason the chat does not know is judged as another one: what was said decides.
func reasonOf(reason supportv1.ReportReason) judgements.Reason {
	switch reason {
	case supportv1.ReportReason_REPORT_REASON_CHEATING:
		return judgements.ReasonCheating
	case supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE:
		return judgements.ReasonChatAbuse
	case supportv1.ReportReason_REPORT_REASON_BAD_NAME:
		return judgements.ReasonBadName
	case supportv1.ReportReason_REPORT_REASON_OTHER, supportv1.ReportReason_REPORT_REASON_UNSPECIFIED:
	}
	return judgements.ReasonOther
}
