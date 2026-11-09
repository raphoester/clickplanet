package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
)

var (
	both    = []calendar.Platform{calendar.TikTok, calendar.YouTube}
	tiktok  = []calendar.Platform{calendar.TikTok}
	youtube = []calendar.Platform{calendar.YouTube}

	// Every hour of the day, an hour apart.
	roomy = calendar.Calendar{Spacing: time.Hour, Budget: calendar.Budget{
		calendar.TikTok: {PerDay: 10}, calendar.YouTube: {PerDay: 8},
	}}
	// The game's.
	paced = calendar.Default
)

// count postings an hour apart from the first, on the platforms.
func postings(count int, first time.Time, platforms ...calendar.Platform) calendar.Feed {
	feed := make(calendar.Feed, 0, count)
	for i := range count {
		feed = append(feed, calendar.Posting{At: first.Add(time.Duration(i) * time.Hour), Platforms: platforms})
	}
	return feed
}

func slotOf(t *testing.T, c calendar.Calendar, platforms []calendar.Platform, feed calendar.Feed, now time.Time) calendar.Slot {
	t.Helper()

	slot, found := c.Slot(platforms, feed, time.Time{}, now)
	require.True(t, found)
	return slot
}

func TestABudgetIsCountedOverATrailingDay(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	// Ten in the day, the first a day old to the minute.
	full := postings(10, now.Add(-24*time.Hour), calendar.TikTok)

	assert.Equal(t, 0, roomy.Left(calendar.TikTok, full, now))
	assert.Equal(t, 1, roomy.Left(calendar.TikTok, full, now.Add(time.Minute)))
}

func TestAPlatformCountsItsOwnPostsAlone(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	full := postings(10, now.Add(-20*time.Hour), calendar.TikTok)

	assert.Equal(t, 0, roomy.Left(calendar.TikTok, full, now))
	assert.Equal(t, 8, roomy.Left(calendar.YouTube, full, now))
}

func TestAPostStillToComeCountsInTheTrailingDayThatEndsWithIt(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	// Five in the day and five still to go out: the day that ends with the last of them holds all ten, and now.
	feed := append(postings(5, now.Add(-19*time.Hour), calendar.TikTok), postings(5, now.Add(time.Hour), calendar.TikTok)...)

	assert.Equal(t, 0, roomy.Left(calendar.TikTok, feed, now))
}

func TestAPostMoreThanADayAheadLeavesTheRoomOfNowAlone(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	// A full day, all of it scheduled ahead: no trailing day that holds now holds any of it.
	ahead := postings(10, now.Add(24*time.Hour+time.Minute), calendar.TikTok)

	assert.Equal(t, 10, roomy.Left(calendar.TikTok, ahead, now))
}

func TestAPlatformTheBudgetDoesNotNameTakesNothing(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)

	assert.Equal(t, 0, calendar.Calendar{}.Left(calendar.TikTok, nil, now))
	_, found := calendar.Calendar{}.Slot(both, nil, time.Time{}, now)
	assert.False(t, found)
}

func TestWithNothingAroundAPostGoesOutTheLeadFromNowOnAWholeMinute(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 53, 27, 500, time.UTC)

	slot := slotOf(t, roomy, both, nil, now)

	assert.Equal(t, calendar.Slot{At: time.Date(2026, 10, 24, 12, 59, 0, 0, time.UTC), Platforms: both}, slot)
}

func TestAPostGoesOutNoSoonerThanItsNotBeforeOnAWholeMinute(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 40, 27, 500, time.UTC)

	slot, found := roomy.Slot(both, nil, now.Add(4*time.Hour), now)

	assert.True(t, found)
	assert.Equal(t, time.Date(2026, 10, 12, 16, 41, 0, 0, time.UTC), slot.At)
}

func TestAPostKeepsASpacingFromThePostsOnItsPlatformsWhoeverMadeThem(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	feed := calendar.Feed{
		{At: now.Add(-3 * time.Hour), Platforms: both},
		// The second post of a run before, still to go out on TikTok.
		{At: now.Add(20 * time.Minute), Platforms: tiktok},
	}

	assert.Equal(t, now.Add(80*time.Minute), slotOf(t, roomy, both, feed, now).At)
}

func TestAPostSoonAfterTheLastOneWaitsOnlyForTheSpacing(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	feed := calendar.Feed{{At: now.Add(-50 * time.Minute), Platforms: both}}

	assert.Equal(t, now.Add(10*time.Minute), slotOf(t, roomy, both, feed, now).At)
}

func TestASecondPostGoesASpacingAfterTheFirst(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 53, 0, 0, time.UTC)
	first := slotOf(t, roomy, both, nil, now)

	second := slotOf(t, roomy, both, calendar.Feed{}.With(first.At, first.Platforms), now)

	assert.Equal(t, time.Date(2026, 10, 24, 12, 58, 0, 0, time.UTC), first.At)
	assert.Equal(t, time.Date(2026, 10, 24, 13, 58, 0, 0, time.UTC), second.At)
}

func TestAPlatformWhoseDayIsFullIsLeftOutOfTheSlot(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	feed := postings(8, now.Add(-23*time.Hour), calendar.TikTok, calendar.YouTube)

	assert.Equal(t, tiktok, slotOf(t, roomy, both, feed, now).Platforms)
}

func TestAPostWithNoPlatformLeftHasNoSlot(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 53, 0, 0, time.UTC)
	feed := append(postings(8, now.Add(-23*time.Hour), both...), postings(2, now.Add(-2*time.Hour), calendar.TikTok)...)

	_, found := roomy.Slot(both, feed, time.Time{}, now)

	assert.False(t, found)
}

func TestAPlatformOutOfItsHoursIsLeftOutOfTheSlot(t *testing.T) {
	assert.Equal(t, youtube, slotOf(t, paced, both, nil, time.Date(2026, 10, 27, 12, 53, 0, 0, time.UTC)).Platforms)
}

func TestASlotInAPlatformsHoursTakesIt(t *testing.T) {
	slot := slotOf(t, paced, both, nil, time.Date(2026, 10, 27, 13, 55, 0, 0, time.UTC))

	assert.Equal(t, calendar.Slot{At: time.Date(2026, 10, 27, 14, 0, 0, 0, time.UTC), Platforms: both}, slot)
}

func TestAPlatformLeftOutForItsHoursNoLongerHoldsTheSlotBack(t *testing.T) {
	now := time.Date(2026, 10, 27, 19, 0, 0, 0, time.UTC)
	// A post by hand on TikTok: its spacing would push the post past TikTok's hours.
	feed := calendar.Feed{{At: now.Add(30 * time.Minute), Platforms: tiktok}}

	assert.Equal(t, calendar.Slot{At: now.Add(5 * time.Minute), Platforms: youtube}, slotOf(t, paced, both, feed, now))
}

func TestAPostOutOfTheHoursOfItsOnlyPlatformHasNoSlot(t *testing.T) {
	_, found := paced.Slot(tiktok, nil, time.Time{}, time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC))

	assert.False(t, found)
}

// A post scheduled ahead, by hand or a battle's teaser, is a reservation: it holds back no post before it.
func TestAPostScheduledAheadHoldsBackNoSlotBeforeIt(t *testing.T) {
	now := time.Date(2026, 11, 12, 14, 52, 0, 0, time.UTC)

	for _, ahead := range []time.Time{
		time.Date(2026, 11, 12, 18, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 13, 18, 0, 0, 0, time.UTC),
	} {
		slot := slotOf(t, paced, both, calendar.Feed{{At: ahead, Platforms: both}}, now)

		assert.Equal(t, calendar.Slot{At: time.Date(2026, 11, 12, 14, 57, 0, 0, time.UTC), Platforms: both}, slot,
			"with a post at %s", ahead)
	}
}

func TestAPostGoesInTheFirstGapThatKeepsTheSpacingOnBothSides(t *testing.T) {
	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	// A post at 10:30, and one by hand at 12:30: 11:30 is an hour from both.
	feed := calendar.Feed{{At: now.Add(30 * time.Minute), Platforms: both}, {At: now.Add(150 * time.Minute), Platforms: youtube}}

	assert.Equal(t, now.Add(90*time.Minute), slotOf(t, roomy, both, feed, now).At)
}

func TestAPostSkipsAGapTooShortForTheSpacingOnBothSides(t *testing.T) {
	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	// 11:30 is an hour after the post at 10:30 and half an hour before the one at 12:00: the post goes an hour after that.
	feed := calendar.Feed{{At: now.Add(30 * time.Minute), Platforms: both}, {At: now.Add(2 * time.Hour), Platforms: youtube}}

	assert.Equal(t, now.Add(3*time.Hour), slotOf(t, roomy, both, feed, now).At)
}

func TestAPostThatWouldOverfillTheDayOfAPostScheduledAheadLeavesItsPlatformOut(t *testing.T) {
	now := time.Date(2026, 11, 12, 18, 52, 0, 0, time.UTC)
	// Two today, and one tomorrow at 14:30: the trailing day that ends with it holds all three.
	feed := calendar.Feed{
		{At: time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 11, 12, 17, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 11, 13, 14, 30, 0, 0, time.UTC), Platforms: tiktok},
	}

	assert.Equal(t, calendar.Slot{At: now.Add(5 * time.Minute), Platforms: youtube}, slotOf(t, paced, both, feed, now))
}

func TestAPostScheduledAheadCountsOnlyInTheTrailingDaysThatHoldIt(t *testing.T) {
	now := time.Date(2026, 11, 12, 18, 52, 0, 0, time.UTC)
	// Two today, and one tomorrow at 15:30: the day that ends with it starts after today's first.
	feed := calendar.Feed{
		{At: time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 11, 12, 17, 0, 0, 0, time.UTC), Platforms: both},
		{At: time.Date(2026, 11, 13, 15, 30, 0, 0, time.UTC), Platforms: tiktok},
	}

	assert.Equal(t, both, slotOf(t, paced, both, feed, now).Platforms)
}

func TestASlotTheCalendarGaveIsOpen(t *testing.T) {
	now := time.Date(2026, 11, 12, 14, 52, 0, 0, time.UTC)
	feed := calendar.Feed{{At: now.Add(-50 * time.Minute), Platforms: both}}

	assert.True(t, paced.Open(slotOf(t, paced, both, feed, now), feed, now))
}

func TestASlotIsNoLongerOpenOnceAPostCrowdsItOrFillsItsDay(t *testing.T) {
	now := time.Date(2026, 11, 12, 14, 52, 0, 0, time.UTC)
	slot := slotOf(t, paced, both, nil, now)

	assert.False(t, paced.Open(slot, calendar.Feed{{At: slot.At.Add(30 * time.Minute), Platforms: youtube}}, now))
	assert.False(t, paced.Open(slot, postings(3, now.Add(-20*time.Hour), calendar.TikTok), now))
	assert.True(t, paced.Open(slot, calendar.Feed{{At: slot.At.Add(time.Hour), Platforms: youtube}}, now))
}

func TestASlotTheLeadHasPassedOrOffAWholeMinuteOrOnNoPlatformIsNotOpen(t *testing.T) {
	now := time.Date(2026, 11, 12, 14, 52, 0, 0, time.UTC)
	slot := slotOf(t, paced, both, nil, now)

	assert.False(t, paced.Open(slot, nil, now.Add(time.Minute)))
	assert.False(t, paced.Open(calendar.Slot{At: slot.At.Add(time.Second), Platforms: both}, nil, now))
	assert.False(t, paced.Open(calendar.Slot{At: slot.At}, nil, now))
}

func TestAPlatformIsKnownAndNamedOnce(t *testing.T) {
	platforms, err := calendar.PlatformsOf("youtube", "tiktok")
	require.NoError(t, err)
	assert.Equal(t, []calendar.Platform{calendar.YouTube, calendar.TikTok}, platforms)

	_, err = calendar.PlatformsOf("instagram")
	require.ErrorIs(t, err, calendar.ErrUnknownPlatform)
	_, err = calendar.PlatformsOf("tiktok", "tiktok")
	require.ErrorContains(t, err, "twice")
}
