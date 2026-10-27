package close_ended_rounds_usecase_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/usecases/play_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/play_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_ended_rounds_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/open_own_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type countedWakes struct{ woken int }

func (c *countedWakes) Wake() { c.woken++ }

// Season 0's Final Battle, Sat 31 Oct 2026: a rush entry from 21:00 to 23:00, and the season the gamemaster commands
// ends at 23:00. The last day closes at 21:00, the battle's round of its own ×3 at 23:00, then the season.
func TestTheFinalBattleClosesItsRoundThenTheSeasonTheGamemasterCommanded(t *testing.T) {
	clock := cptime.NewFixedClock(utc(31, 20, 0))
	seasons := calendar.New(calendar.Config{DailyCutoff: 21 * time.Hour})
	store, outbox, territory, wakes := inmemory_round_store.New(), &cpatomic.FakeOutbox{}, &stubTerritory{}, &countedWakes{}
	closing := close_ended_rounds_usecase.New(territory, store,
		close_round_usecase.New(cpatomic.FakeTransactioner{}, store, outbox),
		close_season_usecase.New(cpatomic.FakeTransactioner{}, store, outbox), seasons, clock)

	require.NoError(t, play_season_handler.New(play_season_usecase.New(inmemory_season_store.New(), seasons, wakes)).
		Handle(t.Context(), &scoringv1.PlaySeason{
			Number: 0, StartsAt: timestamppb.New(time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)),
			EndsAt: timestamppb.New(utc(31, 23, 0)), DecidedAt: timestamppb.New(utc(31, 20, 0)),
		}))
	assert.Equal(t, 1, wakes.woken, "the runner reads the season's end again")
	report, err := closing.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, utc(31, 21, 0), report.Next, "the last day is open")

	territory.held = map[rounds.Country]uint32{"fr": 3, "de": 1}
	clock.Advance(time.Hour)
	entry := rounds.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031"))
	require.NoError(t, open_own_round_usecase.New(seasons, store, clock, wakes).Execute(t.Context(), entry, 3, utc(31, 21, 0), utc(31, 23, 0)))
	report, err = closing.Execute(t.Context())
	require.NoError(t, err)
	require.Len(t, report.Closed, 1, "the last day")
	assert.Equal(t, utc(31, 23, 0), report.Next)

	territory.held = map[rounds.Country]uint32{"de": 4, "fr": 1}
	clock.Advance(2 * time.Hour)
	report, err = closing.Execute(t.Context())
	require.NoError(t, err)

	require.Len(t, report.Closed, 1)
	assert.True(t, report.Closed[0].Round.Own)
	assert.Equal(t, uint32(3), report.Closed[0].Round.Times)
	require.Len(t, report.Ended, 1)
	published := outbox.Recorded()
	require.Len(t, published, 3)
	assert.True(t, proto.Equal(&scoringv1.RoundClosed{
		Season: 0, Number: 1, EndedAt: timestamppb.New(utc(31, 21, 0)),
		Results: []*scoringv1.RoundResult{{Country: "fr", Rank: 1, Points: 25}, {Country: "de", Rank: 2, Points: 18}}, Times: 1,
	}, published[0]), "the last day: %v", published[0])
	assert.True(t, proto.Equal(&scoringv1.RoundClosed{
		Season: 0, Number: 1, Finale: true, EndedAt: timestamppb.New(utc(31, 23, 0)),
		Results: []*scoringv1.RoundResult{{Country: "de", Rank: 1, Points: 75}, {Country: "fr", Rank: 2, Points: 54}},
		Own:     true, Times: 3, EntryId: entry.String(),
	}, published[1]), "the battle: %v", published[1])
	assert.True(t, proto.Equal(&scoringv1.SeasonClosed{
		Season: 0, EndedAt: timestamppb.New(utc(31, 23, 0)),
		Table: []*scoringv1.SeasonPlace{
			{Country: "de", Rank: 1, Points: 93, RoundsWon: 1, FinalePoints: 75},
			{Country: "fr", Rank: 2, Points: 79, RoundsWon: 1, FinalePoints: 54},
		},
	}, published[2]), "then the season: %v", published[2])
}
