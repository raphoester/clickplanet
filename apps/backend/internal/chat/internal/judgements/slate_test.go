package judgements_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

var now = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)

func removal(at time.Time, reason string) moderation.Snapshot {
	return moderation.Snapshot{Kind: moderation.Removed, At: at, Reason: moderation.Reason(reason)}
}

func mute(at time.Time, kind mutes.Kind, duration time.Duration, reason string) moderation.Snapshot {
	return moderation.Snapshot{
		Kind: moderation.Muted, At: at, MuteKind: kind, Duration: duration, Reason: moderation.Reason(reason),
	}
}

func next(history []moderation.Snapshot, kind mutes.Kind, severe bool) judgements.Step {
	return judgements.SlateOf(history, kind, now.Add(-judgements.SlateWindow)).Next(severe)
}

func TestTheStepsAreThePolicysTable(t *testing.T) {
	for offences, want := range []judgements.Step{
		{Offence: 1},
		{Offence: 2, Mute: 15 * time.Minute},
		{Offence: 3, Mute: time.Hour},
		{Offence: 4, Mute: 24 * time.Hour},
	} {
		var history []moderation.Snapshot
		for n := 1; n <= offences; n++ {
			history = append(history, removal(now.Add(-time.Hour), "offence "+strconv.Itoa(n)+" (chat): a slur"))
		}
		assert.Equal(t, want, next(history, mutes.Chat, false), "after %d offences", offences)
	}
}

func TestAMuteAndARemovalOfOneOffenceCountOnce(t *testing.T) {
	history := []moderation.Snapshot{
		removal(now.Add(-time.Hour), "offence 2 (chat): a slur"),
		mute(now.Add(-time.Hour), mutes.Chat, 15*time.Minute, "offence 2 (chat): a slur"),
	}

	assert.Equal(t, judgements.Step{Offence: 3, Mute: time.Hour}, next(history, mutes.Chat, false))
}

func TestTheSlateIsWipedAfterSevenDaysWithNoOffence(t *testing.T) {
	history := []moderation.Snapshot{removal(now.Add(-judgements.SlateWindow-time.Minute), "offence 3 (chat): a slur")}

	assert.Equal(t, judgements.Step{Offence: 1}, next(history, mutes.Chat, false))
}

func TestTheNumberAnOffenceCarriesCarriesTheCountPastTheWindow(t *testing.T) {
	history := []moderation.Snapshot{removal(now.Add(-6*24*time.Hour), "offence 2 (chat): spam")}

	assert.Equal(t, judgements.Step{Offence: 3, Mute: time.Hour}, next(history, mutes.Chat, false),
		"the 2nd was counted from the 1st, a few days before it")
}

func TestTheChatAndTheGifsAreCountedApart(t *testing.T) {
	history := []moderation.Snapshot{
		removal(now.Add(-time.Hour), "offence 3 (gifs): gore"),
		mute(now.Add(-time.Hour), mutes.Gifs, time.Hour, "offence 3 (gifs): gore"),
	}

	assert.Equal(t, judgements.Step{Offence: 1}, next(history, mutes.Chat, false))
	assert.Equal(t, judgements.Step{Offence: 4, Mute: 24 * time.Hour}, next(history, mutes.Gifs, false))
}

func TestASevereOffenceGoesStraightToTheThirdStep(t *testing.T) {
	assert.Equal(t, judgements.Step{Offence: 3, Mute: time.Hour}, next(nil, mutes.Chat, true))

	history := []moderation.Snapshot{removal(now.Add(-time.Hour), "offence 3 (chat): a Nazi slogan")}
	assert.Equal(t, judgements.Step{Offence: 4, Mute: 24 * time.Hour}, next(history, mutes.Chat, false),
		"the next offence is the 24-hour one")
	assert.Equal(t, judgements.Step{Offence: 4, Mute: 24 * time.Hour}, next(history, mutes.Chat, true),
		"never lighter than the count")
}

func TestAfterADayMuteTheOperatorDecides(t *testing.T) {
	afterFourth := []moderation.Snapshot{mute(now.Add(-time.Hour), mutes.Chat, 24*time.Hour, "offence 4 (chat): a threat")}
	byTheOperator := []moderation.Snapshot{mute(now.Add(-time.Hour), mutes.Chat, 48*time.Hour, "operator: a raid")}

	assert.Equal(t, judgements.Step{Offence: 5, ForOperator: true}, next(afterFourth, mutes.Chat, false))
	assert.Equal(t, judgements.Step{Offence: 1, ForOperator: true}, next(byTheOperator, mutes.Chat, false))
	assert.Equal(t, judgements.Step{Offence: 1}, next(byTheOperator, mutes.Gifs, false), "a mute of the chat is not one of the GIFs")
}

func TestNoStepMutesLongerThanADay(t *testing.T) {
	for offences := range 10 {
		history := []moderation.Snapshot{removal(now.Add(-time.Hour), "offence "+strconv.Itoa(offences)+" (chat): spam")}
		assert.LessOrEqual(t, next(history, mutes.Chat, true).Mute, judgements.MaxMute)
	}
}

func TestAnOperatorsReasonIsNoOffence(t *testing.T) {
	history := []moderation.Snapshot{
		removal(now.Add(-time.Hour), "operator: asked by the player"),
		removal(now.Add(-time.Hour), "spam, no offence: a link"),
		removal(now.Add(-time.Hour), ""),
	}

	assert.Equal(t, judgements.Step{Offence: 1}, next(history, mutes.Chat, false))
}
