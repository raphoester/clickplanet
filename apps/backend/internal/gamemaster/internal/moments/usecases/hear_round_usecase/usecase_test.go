package hear_round_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_round_usecase"
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

type harness struct {
	t       *testing.T
	store   *inmemory_moment_store.Store
	outbox  *cpatomic.FakeOutbox
	useCase *hear_round_usecase.Hearing
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	seasons, entries := inmemory_season_store.New(), inmemory_entry_store.New()
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{
		Number: 0, Length: 23*day + 23*time.Hour, Finale: season.Finale{Length: 2 * time.Hour, Set: "rush", Times: 3},
	}.Restore()))
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: early, Set: "rush", Kind: schedule.Battle, StartsAt: finale.Add(-day), EndsAt: finale.Add(-day + time.Hour), Times: 2,
		MadeAt: finale.Add(-2 * day),
	}.Restore()))

	h := &harness{t: t, store: inmemory_moment_store.New(), outbox: &cpatomic.FakeOutbox{}}
	h.useCase = hear_round_usecase.New(moments.NewPlanner(seasons, entries, inmemory_moment_store.New(), season.Cutoff(21*time.Hour)), h.store,
		outbox_moment_events.New(h.outbox), cpatomic.FakeTransactioner{})
	return h
}

func (h *harness) hear(round moments.RoundID, endedAt time.Time) hear_round_usecase.Heard {
	h.t.Helper()
	heard, err := h.useCase.Execute(h.t.Context(), moments.ScoredRound{Round: round, Book: "season-0", EndedAt: endedAt})
	require.NoError(h.t, err)
	return heard
}

func TestADayScoredIsToldOnceAsTheDayItWas(t *testing.T) {
	h := newHarness(t)

	assert.Equal(t, hear_round_usecase.Told, h.hear("season-0/day-24", finale))
	assert.Equal(t, hear_round_usecase.Repeated, h.hear("season-0/day-24", finale), "scoring may tell it twice")

	recorded := h.outbox.Recorded()
	require.Len(t, recorded, 1)
	assert.True(t, proto.Equal(&gamemasterv1.DayEnded{
		Season: 0, Day: 24, RoundId: "season-0/day-24", EndedAt: timestamppb.New(finale),
	}, recorded[0]), "%v", recorded[0])
}

func TestTheFinaleScoredIsToldAsTheSeasonsSecondBattleThatEndsIt(t *testing.T) {
	h := newHarness(t)

	assert.Equal(t, hear_round_usecase.Told, h.hear("season-0/finale", seasonZeroEnds))

	recorded := h.outbox.Recorded()
	require.Len(t, recorded, 1)
	assert.True(t, proto.Equal(&gamemasterv1.BattleEnded{
		Season: 0, Number: 2, Times: 3, EndsSeason: true, RoundId: "season-0/finale", EndedAt: timestamppb.New(seasonZeroEnds),
	}, recorded[0]), "%v", recorded[0])
}

func TestABattleOfAnEntryScoredIsNoFinale(t *testing.T) {
	h := newHarness(t)

	assert.Equal(t, hear_round_usecase.Told, h.hear(moments.RoundID(early.String()), finale.Add(-day+time.Hour)))

	recorded := h.outbox.Recorded()
	require.Len(t, recorded, 1)
	assert.True(t, proto.Equal(&gamemasterv1.BattleEnded{
		Season: 0, Number: 1, EntryId: early.String(), Times: 2, RoundId: early.String(),
		EndedAt: timestamppb.New(finale.Add(-day + time.Hour)),
	}, recorded[0]), "%v", recorded[0])
}

func TestARoundNoDayOrBattleHasIsDroppedAndKeptNot(t *testing.T) {
	h := newHarness(t)

	for _, round := range []moments.RoundID{"season-0/day-25", "season-0/own-1", "season-1/finale", "0192f5a0-7c00-7000-8000-000000000099"} {
		assert.Equal(t, hear_round_usecase.Unknown, h.hear(round, finale), "%q", round)
	}

	assert.Empty(t, h.outbox.Recorded())
	kept, err := h.store.KeepRound(t.Context(), moments.ScoredRound{Round: "season-0/day-25", Book: "season-0", EndedAt: finale})
	require.NoError(t, err)
	assert.True(t, kept)
}

func TestARoundThatCannotBeKeptIsAnErrorAndToldNot(t *testing.T) {
	h := newHarness(t)
	down := errors.New("postgres is down")
	h.store.FailWith(down)

	_, err := h.useCase.Execute(t.Context(), moments.ScoredRound{Round: "season-0/day-1", Book: "season-0", EndedAt: finale})

	require.ErrorIs(t, err, down)
	assert.Empty(t, h.outbox.Recorded())
}
