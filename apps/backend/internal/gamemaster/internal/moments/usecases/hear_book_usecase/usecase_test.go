package hear_book_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

const day = 24 * time.Hour

var (
	finale         = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonZeroEnds = finale.Add(2 * time.Hour)
	early          = schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000001"))
)

func newHearing(t *testing.T, store *inmemory_moment_store.Store, outbox *cpatomic.FakeOutbox) *hear_book_usecase.Hearing {
	t.Helper()

	seasons, entries := inmemory_season_store.New(), inmemory_entry_store.New()
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{
		Number: 0, Length: 23*day + 23*time.Hour, Finale: season.Finale{Length: 2 * time.Hour, Set: "rush", Times: 3},
	}.Restore()))
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: early, Set: "rush", Kind: schedule.Battle, StartsAt: finale.Add(-time.Hour), EndsAt: finale, Times: 2, MadeAt: finale.Add(-day),
	}.Restore()))
	return hear_book_usecase.New(moments.NewPlanner(seasons, entries, inmemory_moment_store.New(), season.Cutoff(21*time.Hour)), store,
		outbox_moment_events.New(outbox), cpatomic.FakeTransactioner{})
}

func TestAClosedBookIsToldOnceAsTheSeasonsEndWithItsFinale(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}
	hearing := newHearing(t, inmemory_moment_store.New(), outbox)
	closed := moments.ClosedBook{Book: "season-0", EndedAt: seasonZeroEnds}

	heard, err := hearing.Execute(t.Context(), closed)
	require.NoError(t, err)
	assert.Equal(t, hear_book_usecase.Told, heard)
	heard, err = hearing.Execute(t.Context(), closed)
	require.NoError(t, err)
	assert.Equal(t, hear_book_usecase.Repeated, heard)

	recorded := outbox.Recorded()
	require.Len(t, recorded, 1)
	assert.True(t, proto.Equal(&gamemasterv1.SeasonEnded{
		Season: 0, BookId: "season-0", EndedAt: timestamppb.New(seasonZeroEnds), FinaleRoundId: "season-0/finale",
		FinaleBattle: &gamemasterv1.BattleEnded{
			Season: 0, Number: 2, Times: 3, EndsSeason: true, RoundId: "season-0/finale", EndedAt: timestamppb.New(seasonZeroEnds),
		},
	}, recorded[0]), "%v", recorded[0])
}

func TestABookNoSeasonHasIsDropped(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}
	hearing := newHearing(t, inmemory_moment_store.New(), outbox)

	for _, book := range []moments.BookID{"season-1", "season-0/day-1", ""} {
		heard, err := hearing.Execute(t.Context(), moments.ClosedBook{Book: book, EndedAt: seasonZeroEnds})
		require.NoError(t, err)
		assert.Equal(t, hear_book_usecase.Unknown, heard, "%q", book)
	}
	assert.Empty(t, outbox.Recorded())
}

func TestABookThatCannotBeKeptIsAnErrorAndToldNot(t *testing.T) {
	store, outbox := inmemory_moment_store.New(), &cpatomic.FakeOutbox{}
	down := errors.New("postgres is down")
	store.FailWith(down)

	_, err := newHearing(t, store, outbox).Execute(t.Context(), moments.ClosedBook{Book: "season-0", EndedAt: seasonZeroEnds})

	require.ErrorIs(t, err, down)
	assert.Empty(t, outbox.Recorded())
}
