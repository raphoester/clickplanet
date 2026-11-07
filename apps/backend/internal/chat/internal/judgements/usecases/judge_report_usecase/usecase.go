package judge_report_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase/publishing_mute"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/usecases/remove_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Judgements interface {
	OfReport(ctx context.Context, report judgements.ReportID) (judgements.Judgement, error)
	OfMessage(ctx context.Context, message messages.MessageID) (judgements.Judgement, error)
	Save(ctx context.Context, judgement judgements.Judgement) error
}

type Conversations interface {
	// messages.ErrNotKept for a message named that is not kept since then.
	Conversation(
		ctx context.Context, message messages.MessageID, account messages.AccountID, since time.Time,
	) (judgements.Conversation, error)
}

type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

type History interface {
	ActionsOn(ctx context.Context, account messages.AccountID, since time.Time) ([]moderation.Snapshot, error)
}

type Judge interface {
	// judgements.ErrNoJudge with no key, judgements.ErrUnjudgeable for what it would never judge.
	Judge(ctx context.Context, report judgements.Report, conversation judgements.Conversation) (judgements.Answer, error)
}

// The hourly cap on what the judge costs: a report flood is left to the operator once it is spent.
type Calls interface {
	Take(key string) (bool, cpratelimit.State)
}

type Remover interface {
	Execute(ctx context.Context, in remove_message_usecase.In) (time.Time, error)
}

type Muter interface {
	Execute(ctx context.Context, in mute_usecase.In) (mutes.Mute, error)
}

// Every port the judge needs, named: it reads, asks, then acts.
type Ports struct {
	Judgements    Judgements
	Conversations Conversations
	Authors       Authors
	History       History
	Judge         Judge
	Calls         Calls
	Transactioner cpatomic.Transactioner
	Remover       Remover
	Muter         Muter
	Outbox        cpatomic.Outbox
}

type Out struct {
	Judgement judgements.Judgement
	// Told again: the judgement kept the first time, and nothing done.
	Again bool
	// What the model cost, when it was asked.
	Spend judgements.Spend
}

type Executor interface {
	Execute(ctx context.Context, report judgements.Report) (Out, error)
}

const callsKey = "judge"

func New(ports Ports, window messages.Window, clock cptime.Clock) *UseCase {
	return &UseCase{ports: ports, window: window, clock: clock}
}

type UseCase struct {
	ports  Ports
	window messages.Window
	clock  cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// The cheap reads first, and the model only once nothing else decided the report.
func (u *UseCase) Execute(ctx context.Context, report judgements.Report) (Out, error) {
	kept, err := u.ports.Judgements.OfReport(ctx, report.ID)
	switch {
	case err == nil:
		return Out{Judgement: kept, Again: true}, nil
	case !errors.Is(err, judgements.ErrNotJudged):
		return Out{}, fmt.Errorf("failed to read whether the report was judged: %w", err)
	}

	now := u.clock.Now()
	switch report.Reason {
	case judgements.ReasonCheating:
		return u.left(ctx, report, false, true, "cheating: the antibot's to inspect, and the operator's to decide", now)
	case judgements.ReasonBadName:
		return u.left(ctx, report, false, false, "a name: the operator's to rename", now)
	case judgements.ReasonChatAbuse, judgements.ReasonOther:
	}

	if report.Message != judgements.NoMessage {
		earlier, err := u.ports.Judgements.OfMessage(ctx, report.Message)
		switch {
		case err == nil:
			return u.decided(ctx, judgements.Decision{Judgement: earlier.Followed(report, now)})
		case !errors.Is(err, judgements.ErrNotJudged):
			return Out{}, fmt.Errorf("failed to read whether the message was judged: %w", err)
		}
	}

	conversation, err := u.ports.Conversations.Conversation(ctx, report.Message, report.Reported, u.window.Since(now))
	switch {
	case errors.Is(err, messages.ErrNotKept):
		return u.decided(ctx, judgements.Decided(report, judgements.Dismissed, true, false,
			"the message is past the chat's retention, or was never kept", now))
	case err != nil:
		return Out{}, fmt.Errorf("failed to read the conversation: %w", err)
	}
	if reported := conversation.Reported; reported != nil {
		switch {
		case reported.Account != report.Reported:
			return u.left(ctx, report, false, true, "the message reported is not the reported player's", now)
		case reported.Removed:
			return u.decided(ctx, judgements.Decided(report, judgements.Acted, true, false, "the message was removed already", now))
		}
	}
	if conversation.Empty() {
		return u.decided(ctx, judgements.Decided(report, judgements.Dismissed, false, false,
			"nothing the player said is kept", now))
	}

	author, err := u.ports.Authors.Author(ctx, report.Reported)
	if err != nil {
		return Out{}, fmt.Errorf("failed to ask who the reported player is: %w", err)
	}
	if author.Admin() {
		return u.left(ctx, report, false, false, "the operator's own account is never acted on", now)
	}

	if allowed, _ := u.ports.Calls.Take(callsKey); !allowed {
		return u.left(ctx, report, false, true, "the hourly cap of judge calls is spent", now)
	}

	history, err := u.ports.History.ActionsOn(ctx, report.Reported, now.Add(-judgements.SlateWindow))
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the player's moderation history: %w", err)
	}

	answer, err := u.ports.Judge.Judge(ctx, report, conversation)
	switch {
	case errors.Is(err, judgements.ErrNoJudge):
		return u.left(ctx, report, false, false, "no judge: no ANTHROPIC_API_KEY", now)
	case errors.Is(err, judgements.ErrUnjudgeable):
		out, failed := u.left(ctx, report, true, true, "unsure: "+err.Error(), now)
		out.Spend = answer.Spend
		return out, failed
	case err != nil:
		return Out{}, fmt.Errorf("failed to judge the report: %w", err)
	}

	out, err := u.decided(ctx, judgements.Decide(report, conversation, answer, history, now))
	out.Spend = answer.Spend
	return out, err
}

func (u *UseCase) left(
	ctx context.Context, report judgements.Report, judged, flagged bool, reason string, at time.Time,
) (Out, error) {
	return u.decided(ctx, judgements.Decided(report, judgements.LeftOpen, judged, flagged, reason, at))
}

// The removals, the mute and its announcement, the judgement and the report's decision: all of it, or none.
func (u *UseCase) decided(ctx context.Context, decision judgements.Decision) (Out, error) {
	judgement := decision.Judgement
	report := judgement.TakeSnapshot()
	err := u.ports.Transactioner.InTx(ctx, func(ctx context.Context) error {
		for _, message := range decision.Remove {
			if _, err := u.ports.Remover.Execute(ctx, remove_message_usecase.In{
				Message: message, Reason: decision.Reason, By: judgements.By,
			}); err != nil {
				return fmt.Errorf("failed to remove %q: %w", message, err)
			}
		}
		if decision.Mute > 0 {
			mute, err := u.ports.Muter.Execute(ctx, mute_usecase.In{
				Account: decision.Account, Kind: decision.Kind, Duration: decision.Mute, Reason: decision.Reason,
				By: judgements.By,
			})
			if err != nil {
				return fmt.Errorf("failed to mute: %w", err)
			}
			if err := u.ports.Outbox.Record(ctx, publishing_mute.EventOf(mute)); err != nil {
				return fmt.Errorf("failed to record the mute's announcement: %w", err)
			}
		}
		if err := u.ports.Judgements.Save(ctx, judgement); err != nil {
			return fmt.Errorf("failed to keep the judgement: %w", err)
		}
		if outcome, closes := outcomeOf(report.Verdict); closes {
			if err := u.ports.Outbox.Record(ctx, &supportv1.DecideReport{
				ReportId: report.Report.String(), Outcome: outcome, Comment: judgement.Comment(),
			}); err != nil {
				return fmt.Errorf("failed to record the report's decision: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return Out{}, fmt.Errorf("failed to act on the report: %w", err)
	}
	return Out{Judgement: judgement}, nil
}

func outcomeOf(verdict judgements.Verdict) (supportv1.ReportOutcome, bool) {
	switch verdict {
	case judgements.Acted:
		return supportv1.ReportOutcome_REPORT_OUTCOME_ACTED, true
	case judgements.Dismissed:
		return supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED, true
	default:
		return supportv1.ReportOutcome_REPORT_OUTCOME_UNSPECIFIED, false
	}
}
