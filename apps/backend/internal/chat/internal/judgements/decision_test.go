package judgements_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

var (
	raider    = messages.AccountID{15: 1}
	reporter  = messages.AccountID{15: 2}
	bystander = messages.AccountID{15: 3}
	reportID  = judgements.ReportID{15: 9}
	gif       = messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 124}
)

// m-0 is the oldest, a minute before m-1, and so on.
func line(id string, account messages.AccountID, text string) judgements.Line {
	sentAt := now.Add(time.Duration(id[len(id)-1]-'9') * time.Minute)
	return judgements.Line{ID: messages.MessageID(id), Account: account, Room: messages.World, SentAt: sentAt, Text: text}
}

func conversation() judgements.Conversation {
	reported := line("m-2", raider, "<a slur>")
	return judgements.Conversation{
		Reported: &reported,
		Before:   []judgements.Line{line("m-1", bystander, "gg France")},
		After:    []judgements.Line{line("m-3", reporter, "reported")},
		Own:      []judgements.Line{line("m-0", raider, "hello"), reported},
	}
}

func report() judgements.Report {
	return judgements.Report{ID: reportID, Reporter: reporter, Reported: raider, Reason: judgements.ReasonChatAbuse, Message: "m-2", At: now}
}

func decide(answer judgements.Answer, history ...moderation.Snapshot) judgements.Decision {
	return judgements.Decide(report(), conversation(), answer, history, now)
}

func TestNoOffenceIsDismissedWithItsReasonAndDoesNothing(t *testing.T) {
	decision := decide(judgements.Answer{Category: judgements.CategoryNone, Why: "banter between rivals"})

	assert.Equal(t, judgements.Snapshot{
		Report: reportID, Message: "m-2", Verdict: judgements.Dismissed, Judged: true,
		Reason: "no offence: banter between rivals", At: now,
	}, decision.Judgement.TakeSnapshot())
	assert.Empty(t, decision.Remove)
	assert.Zero(t, decision.Mute)
	assert.Equal(t, "Claude judge: no offence: banter between rivals", decision.Judgement.Comment())
}

func TestAFirstOffenceRemovesTheMessageAndMutesNobody(t *testing.T) {
	decision := decide(judgements.Answer{
		Category: judgements.CategoryOffence, Offending: []messages.MessageID{"m-2"}, What: "an insult aimed at a player",
	})

	assert.Equal(t, []messages.MessageID{"m-2"}, decision.Remove)
	assert.Zero(t, decision.Mute)
	assert.Equal(t, "offence 1 (chat): an insult aimed at a player", decision.Reason)
	assert.Equal(t, judgements.Acted, decision.Judgement.TakeSnapshot().Verdict)
	assert.Equal(t, "offence 1 (chat): an insult aimed at a player; 1 message removed", decision.Judgement.TakeSnapshot().Reason)
}

func TestASecondOffenceMutesFifteenMinutes(t *testing.T) {
	decision := decide(judgements.Answer{
		Category: judgements.CategoryOffence, Offending: []messages.MessageID{"m-2"}, What: "a threat",
	}, removal(now.Add(-time.Hour), "offence 1 (chat): a threat"))

	assert.Equal(t, 15*time.Minute, decision.Mute)
	assert.Equal(t, mutes.Chat, decision.Kind)
	assert.Equal(t, "offence 2 (chat): a threat", decision.Reason)
	assert.Equal(t, "offence 2 (chat): a threat; 1 message removed, muted 15 minutes", decision.Judgement.TakeSnapshot().Reason)
}

func TestASevereOffenceMutesAnHourAndIsFlagged(t *testing.T) {
	decision := decide(judgements.Answer{
		Category: judgements.CategoryOffence, Severe: true, Offending: []messages.MessageID{"m-2"}, What: "a racial slur",
	})

	assert.Equal(t, time.Hour, decision.Mute)
	assert.Equal(t, "offence 3 (chat): a racial slur", decision.Reason)
	assert.True(t, decision.Judgement.TakeSnapshot().Flagged)
}

func TestOnlyThePlayersOwnMessagesNotRemovedYetAreRemoved(t *testing.T) {
	c := conversation()
	c.Own = append(c.Own, judgements.Line{ID: "m-gone", Account: raider, Room: messages.World, SentAt: now, Removed: true})

	decision := judgements.Decide(report(), c, judgements.Answer{
		Category:  judgements.CategoryOffence,
		Offending: []messages.MessageID{"m-1", "m-2", "m-2", "m-gone", "m-unknown", "m-0"},
	}, nil, now)

	assert.Equal(t, []messages.MessageID{"m-0", "m-2"}, decision.Remove,
		"never a bystander's, never one shown nowhere, never twice")
	assert.Equal(t, "offence 1 (chat): a message against the rules; 2 messages removed", decision.Judgement.TakeSnapshot().Reason)
}

func TestAnOffenceWithNoMessageOfThePlayersIsLeftOpen(t *testing.T) {
	decision := decide(judgements.Answer{Category: judgements.CategoryOffence, Offending: []messages.MessageID{"m-1"}})

	assert.Equal(t, judgements.LeftOpen, decision.Judgement.TakeSnapshot().Verdict)
	assert.True(t, decision.Judgement.TakeSnapshot().Flagged)
	assert.Empty(t, decision.Remove)
}

func TestAGifOffenceTakesTheGifMute(t *testing.T) {
	c := conversation()
	reported := judgements.Line{ID: "m-2", Account: raider, Room: messages.World, SentAt: now, Gif: gif}
	c.Reported, c.Own = &reported, []judgements.Line{reported}

	decision := judgements.Decide(report(), c, judgements.Answer{
		Category: judgements.CategoryOffence, Severe: true, Offending: []messages.MessageID{"m-2"}, What: "gore",
	}, nil, now)

	assert.Equal(t, mutes.Gifs, decision.Kind)
	assert.Equal(t, "offence 3 (gifs): gore", decision.Reason)
	assert.Equal(t, "offence 3 (gifs): gore; 1 message removed, muted 1 hour from the GIFs", decision.Judgement.TakeSnapshot().Reason)
}

func TestAFirstSpamIsRemovedAndIsNoOffence(t *testing.T) {
	decision := decide(judgements.Answer{Category: judgements.CategorySpam, Offending: []messages.MessageID{"m-2"}, What: "a link to a server"})

	assert.Equal(t, "spam, no offence: a link to a server", decision.Reason)
	assert.Zero(t, decision.Mute)
	assert.Equal(t, judgements.Acted, decision.Judgement.TakeSnapshot().Verdict)
}

func TestSpamAgainAfterARemovalIsAnOffence(t *testing.T) {
	decision := decide(judgements.Answer{Category: judgements.CategorySpam, Offending: []messages.MessageID{"m-2"}, What: "the same line"},
		removal(now.Add(-time.Hour), "spam, no offence: the same line"),
		removal(now.Add(-2*time.Hour), "offence 1 (chat): a slur"))

	assert.Equal(t, "offence 2 (chat): the same line", decision.Reason)
	assert.Equal(t, 15*time.Minute, decision.Mute)
}

func TestAfterADayMuteTheMessageGoesAndTheReportIsTheOperators(t *testing.T) {
	decision := decide(judgements.Answer{Category: judgements.CategoryOffence, Offending: []messages.MessageID{"m-2"}, What: "a slur"},
		mute(now.Add(-time.Hour), mutes.Chat, 24*time.Hour, "offence 4 (chat): a slur"))

	assert.Equal(t, judgements.LeftOpen, decision.Judgement.TakeSnapshot().Verdict)
	assert.Equal(t, []messages.MessageID{"m-2"}, decision.Remove)
	assert.Zero(t, decision.Mute, "no step mutes past a day")
	assert.True(t, decision.Judgement.TakeSnapshot().Flagged)
	assert.Contains(t, decision.Judgement.TakeSnapshot().Reason, "the operator decides")
}

func TestWhatIsNeverActedOnIsLeftOpenAndFlagged(t *testing.T) {
	for _, category := range []judgements.Category{
		judgements.CategoryChildSafety, judgements.CategorySelfHarm, judgements.CategoryCheating, judgements.CategoryUnsure, "made-up",
	} {
		decision := decide(judgements.Answer{Category: category, Offending: []messages.MessageID{"m-2"}, Severe: true, Why: "asks a kid for its age"})

		assert.Equal(t, judgements.LeftOpen, decision.Judgement.TakeSnapshot().Verdict, category)
		assert.True(t, decision.Judgement.TakeSnapshot().Flagged, category)
		assert.Empty(t, decision.Remove, category)
		assert.Zero(t, decision.Mute, category)
	}
}

func TestTheModelsWordsAreCutToOneShortLine(t *testing.T) {
	decision := decide(judgements.Answer{
		Category: judgements.CategoryOffence, Offending: []messages.MessageID{"m-2"}, What: "a\nslur " + strings.Repeat("x", 500),
	})

	require.LessOrEqual(t, len([]rune(decision.Reason)), moderation.MaxReasonLength)
	assert.NotContains(t, decision.Reason, "\n")
}

func TestAnotherReportOfAMessageFollowsItsJudgement(t *testing.T) {
	first := decide(judgements.Answer{Category: judgements.CategoryOffence, Offending: []messages.MessageID{"m-2"}, What: "a slur"}).Judgement
	second := judgements.Report{ID: judgements.ReportID{15: 10}, Reported: raider, Message: "m-2"}

	followed := first.Followed(second, now.Add(time.Minute)).TakeSnapshot()

	assert.Equal(t, judgements.Acted, followed.Verdict)
	assert.False(t, followed.Judged, "the first stands for the message")
	assert.Equal(t, "as report "+reportID.String()+": offence 1 (chat): a slur; 1 message removed", followed.Reason)
}
