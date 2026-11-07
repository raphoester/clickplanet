package days_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days/inmemory_day_store"
)

type countingCalendar struct {
	told  *inmemory_day_store.Store
	asked int
	err   error
}

func (c *countingCalendar) DayAt(ctx context.Context, at time.Time) (days.Day, error) {
	c.asked++
	if c.err != nil {
		return days.Day{}, c.err
	}
	return c.told.DayAt(ctx, at) //nolint:wrapcheck // a fake.
}

type fact struct {
	at   time.Time
	flag string
}

func (f fact) of() (time.Time, string) { return f.at, f.flag }

var (
	monday  = days.TestingDayAt(starts)
	tuesday = days.TestingDayAt(starts.Add(24 * time.Hour))
)

func TestABatchIsGroupedByDayAndKeyInTheOrderEachGroupCame(t *testing.T) {
	calendar := &countingCalendar{told: inmemory_day_store.New(monday, tuesday)}
	inMonday, inTuesday := monday.Snapshot().StartsAt.Add(time.Hour), tuesday.Snapshot().StartsAt
	facts := []fact{
		{at: inMonday, flag: "fr"},
		{at: inMonday, flag: "dz"},
		{at: inMonday.Add(time.Minute), flag: "fr"},
		{at: inTuesday, flag: "fr"},
		{at: inMonday.Add(2 * time.Minute), flag: "dz"},
	}

	groups, err := days.GroupsOf(t.Context(), calendar, facts, fact.of)

	require.NoError(t, err)
	assert.Equal(t, []days.Group[string, fact]{
		{Day: monday, Key: "fr", Facts: []fact{facts[0], facts[2]}},
		{Day: monday, Key: "dz", Facts: []fact{facts[1], facts[4]}},
		{Day: tuesday, Key: "fr", Facts: []fact{facts[3]}},
	}, groups)
	assert.Equal(t, 2, calendar.asked, "each day read once")
}

func TestAFactNoDayRunsAtIsLeftOutAndItsTimeAskedOnce(t *testing.T) {
	calendar := &countingCalendar{told: inmemory_day_store.New(monday)}
	between := monday.Snapshot().EndsAt.Add(time.Minute)

	groups, err := days.GroupsOf(t.Context(), calendar,
		[]fact{{at: between, flag: "fr"}, {at: between, flag: "fr"}, {at: between, flag: "dz"}}, fact.of)

	require.NoError(t, err)
	assert.Empty(t, groups)
	assert.Equal(t, 1, calendar.asked, "an enclose's takes share their time")
}

func TestACalendarThatCannotBeReadGroupsNothing(t *testing.T) {
	down := errors.New("down")

	_, err := days.GroupsOf(t.Context(), &countingCalendar{err: down}, []fact{{at: starts, flag: "fr"}}, fact.of)

	assert.ErrorIs(t, err, down)
}
