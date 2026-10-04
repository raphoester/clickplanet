package watch_lead_usecase_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/lead/usecases/watch_lead_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	finaleStarts = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonEnds   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
)

type fakePlanet struct {
	tiles map[string]uint32
	err   error
	reads int
}

func (f *fakePlanet) Shares(context.Context) (lead.Shares, error) {
	f.reads++
	if f.err != nil {
		return lead.Shares{}, f.err
	}
	return lead.SharesOf(maps.Clone(f.tiles)), nil
}

type watch struct {
	clock  *cptime.FixedClock
	planet *fakePlanet
	events *cpbootstrap.RecordedEvents
	*watch_lead_usecase.UseCase
}

func watching(at time.Time, tiles map[string]uint32) watch {
	clock := cptime.NewFixedClock(at)
	planet := &fakePlanet{tiles: tiles}
	events := cpbootstrap.NewRecordedEvents()
	seasons := calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour}}})

	return watch{
		clock:   clock,
		planet:  planet,
		events:  events,
		UseCase: watch_lead_usecase.New(seasons, lead.Config{Margin: 50, Hold: 30 * time.Second}, clock, planet, events),
	}
}

func (w watch) tick(t *testing.T, d time.Duration) {
	t.Helper()

	w.clock.Advance(d)
	require.NoError(t, w.Execute(t.Context()))
}

func TestNothingIsReadBeforeTheFinale(t *testing.T) {
	w := watching(finaleStarts.Add(-time.Hour), map[string]uint32{"fr": 500})

	w.tick(t, time.Second)

	assert.Zero(t, w.planet.reads)
	assert.Empty(t, w.events.Published())
}

func TestALeadTakenAndHeldDuringTheFinaleIsPublishedOnce(t *testing.T) {
	w := watching(finaleStarts, map[string]uint32{"fr": 500, "bg": 400})
	w.tick(t, time.Second)
	require.Empty(t, w.events.Published(), "the leader the finale starts with is no change")

	w.planet.tiles["bg"] = 600
	for range 40 {
		w.tick(t, time.Second)
	}

	published := w.events.Published()
	require.Len(t, published, 1)
	assert.True(t, proto.Equal(&seasonsv1.LeadChanged{
		Season:    0,
		Leader:    "bg",
		Passed:    "fr",
		ChangedAt: timestamppb.New(finaleStarts.Add(32 * time.Second)),
	}, published[0]), "%v", published[0])
}

func TestARestartInTheFinaleNamesTheLeaderSilently(t *testing.T) {
	w := watching(finaleStarts.Add(time.Hour), map[string]uint32{"fr": 500, "bg": 900})

	for range 60 {
		w.tick(t, time.Second)
	}

	assert.Empty(t, w.events.Published())
}

func TestTheEndNamesTheWinnerOnce(t *testing.T) {
	w := watching(seasonEnds.Add(-time.Second), map[string]uint32{"fr": 500, "dz": 900})
	w.tick(t, 0)

	w.tick(t, time.Second)
	w.tick(t, time.Second)

	var ended []proto.Message
	for _, event := range w.events.Published() {
		if _, ok := event.(*seasonsv1.SeasonEnded); ok {
			ended = append(ended, event)
		}
	}
	require.Len(t, ended, 1)
	assert.True(t, proto.Equal(&seasonsv1.SeasonEnded{
		Season: 0, Winner: "dz", EndedAt: timestamppb.New(seasonEnds),
	}, ended[0]))
}

func TestAFailedReadAtTheEndIsTriedAgain(t *testing.T) {
	w := watching(seasonEnds.Add(time.Second), map[string]uint32{"dz": 900})
	w.planet.err = errors.New("planet is not listening yet")

	require.Error(t, w.Execute(t.Context()))
	assert.Empty(t, w.events.Published())

	w.planet.err = nil
	w.tick(t, time.Second)
	require.Len(t, w.events.Published(), 1)
}

func TestABootLongAfterTheEndSaysNothing(t *testing.T) {
	w := watching(seasonEnds.Add(time.Hour), map[string]uint32{"dz": 900})

	w.tick(t, time.Second)

	assert.Empty(t, w.events.Published())
	assert.Zero(t, w.planet.reads)
}

func TestAnEmptyMapHasNoWinner(t *testing.T) {
	w := watching(seasonEnds, map[string]uint32{})

	w.tick(t, time.Second)
	w.tick(t, time.Second)

	assert.Empty(t, w.events.Published())
	assert.Equal(t, 1, w.planet.reads, "an end with nobody on the map is told once, as nothing")
}
