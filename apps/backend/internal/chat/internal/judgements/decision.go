package judgements

import (
	"fmt"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

// What a judgement does: the messages to remove and the mute, each with the reason, and the judgement kept.
type Decision struct {
	Judgement Judgement
	Remove    []messages.MessageID
	// Who is muted: the reported player.
	Account messages.AccountID
	Kind    mutes.Kind
	// 0 for no mute.
	Mute time.Duration
	// The history's: "offence 2 (chat): a slur", "spam, no offence: a link".
	Reason string
}

// A report decided by what became of it, with nothing removed nor muted.
func Decided(report Report, verdict Verdict, judged, flagged bool, reason string, at time.Time) Decision {
	return Decision{Judgement: Judgement{snapshot: Snapshot{
		Report: report.ID, Message: report.Message, Verdict: verdict, Judged: judged, Flagged: flagged, Reason: reason,
		At: at,
	}}}
}

// The policy's steps, applied to what the model answered: the model says what was said, and this says what it costs.
// history is the reported account's, from SlateWindow ago.
func Decide(report Report, conversation Conversation, answer Answer, history []moderation.Snapshot, now time.Time) Decision {
	why := whyOf(answer.Why)
	switch answer.Category {
	case CategoryNone:
		return Decided(report, Dismissed, true, false, joined("no offence", why), now)
	case CategoryOffence, CategorySpam:
	case CategoryChildSafety, CategorySelfHarm, CategoryCheating, CategoryUnsure:
		return Decided(report, LeftOpen, true, true, joined(string(answer.Category), why), now)
	default:
		return Decided(report, LeftOpen, true, true, joined("unsure: the model answered "+string(answer.Category), why), now)
	}

	lines := conversation.offending(answer.Offending, report.Reported)
	if len(lines) == 0 {
		return Decided(report, LeftOpen, true, true, joined("unsure: no message of the player's to remove", why), now)
	}
	kind := kindOf(lines)
	slate := SlateOf(history, kind, now.Add(-SlateWindow))
	what := whatOf(answer.What)
	removed := idsOf(lines)

	if answer.Category == CategorySpam && !slate.spammed {
		reason := firstSpam + ": " + what
		return Decision{
			Judgement: judged(report, Acted, false, reason+"; "+removedOf(removed), now),
			Remove:    removed, Account: report.Reported, Kind: kind, Reason: reason,
		}
	}

	step := slate.Next(answer.Severe)
	reason := fmt.Sprintf("offence %d (%s): %s", step.Offence, kind, what)
	if step.ForOperator {
		return Decision{
			Judgement: judged(report, LeftOpen, true, reason+"; "+removedOf(removed)+"; after a 24-hour mute, the operator decides", now),
			Remove:    removed, Account: report.Reported, Kind: kind, Reason: reason,
		}
	}
	done := removedOf(removed)
	if step.Mute > 0 {
		done += ", muted " + durationOf(step.Mute)
		if kind == mutes.Gifs {
			done += " from the GIFs"
		}
	}
	return Decision{
		Judgement: judged(report, Acted, answer.Severe, reason+"; "+done, now),
		Remove:    removed, Account: report.Reported, Kind: kind, Mute: step.Mute, Reason: reason,
	}
}

func judged(report Report, verdict Verdict, flagged bool, reason string, at time.Time) Judgement {
	return Judgement{snapshot: Snapshot{
		Report: report.ID, Message: report.Message, Verdict: verdict, Judged: true, Flagged: flagged, Reason: reason, At: at,
	}}
}

// A GIF offence takes the GIF mute, so the player keeps its text; anything with a text takes the chat's.
func kindOf(lines []Line) mutes.Kind {
	for _, line := range lines {
		if line.Gif.None() {
			return mutes.Chat
		}
	}
	return mutes.Gifs
}

func idsOf(lines []Line) []messages.MessageID {
	ids := make([]messages.MessageID, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.ID)
	}
	return ids
}

func removedOf(ids []messages.MessageID) string {
	if len(ids) == 1 {
		return "1 message removed"
	}
	return fmt.Sprintf("%d messages removed", len(ids))
}

func durationOf(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return "24 hours"
	case d >= time.Hour:
		return "1 hour"
	default:
		return fmt.Sprintf("%d minutes", int(d/time.Minute))
	}
}

func joined(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, ": ")
}
