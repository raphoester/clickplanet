package turn_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_turn_handler/turn_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/postgres_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/postgres_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/postgres_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	finale     = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	battleEnds = finale.Add(2 * time.Hour)
	madeAt     = finale.Add(-time.Hour)
	seasonZero = &gamemasterv1.Season{
		Number: 0, StartsAt: timestamppb.New(time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)), EndsAt: timestamppb.New(battleEnds),
		BookId: "season-0",
	}
	rush    = uuid.UUID{15: 1}.String()
	dayLast = &gamemasterv1.Day{
		Number: 24, StartsAt: timestamppb.New(finale.Add(-24 * time.Hour)), EndsAt: timestamppb.New(finale), RoundId: "season-0/day-24",
	}
)

func battle(t *testing.T, db cppg.Conn) {
	t.Helper()
	save(t, db, schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 1}), Set: "rush", Kind: schedule.Battle, Description: "Triple points.",
		StartsAt: finale, EndsAt: battleEnds, Priority: 1, Times: 3, MadeAt: madeAt,
	})
}

func save(t *testing.T, db cppg.Conn, entry schedule.EntrySnapshot) {
	t.Helper()
	require.NoError(t, postgres_entry_store.New(db).Save(t.Context(), entry.Restore()))
}

func nowAt(t *testing.T, db cppg.Conn, at time.Time) *gamemasterv1.GetTurnResponse {
	t.Helper()
	answer, err := turn_query.NewPostgresQuery(db, cptime.NewFixedClock(at), season.Cutoff(21*time.Hour)).Now(t.Context())
	require.NoError(t, err)
	return answer
}

func turnAt(t *testing.T, db cppg.Conn, at time.Time) *gamemasterv1.Turn {
	t.Helper()
	return nowAt(t, db, at).GetTurn()
}

func TestWhileABattleRunsTheTurnIsItsIdItsKindItsWordsItsEndAndItsRound(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, finale.Add(time.Minute))

	assert.True(t, proto.Equal(&gamemasterv1.Turn{
		Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, Description: "Triple points.", Since: timestamppb.New(finale),
		EndsAt: timestamppb.New(battleEnds), EntryId: rush, RoundId: rush, Times: 3,
	}, answer), "%v", answer)
}

func TestWithNoEntryRunningTheTurnIsPlainSinceTheLastEnd(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, battleEnds)

	assert.True(t, proto.Equal(&gamemasterv1.Turn{
		Kind: gamemasterv1.EntryKind_ENTRY_KIND_PLAIN, Since: timestamppb.New(battleEnds),
	}, answer), "%v", answer)
}

func TestBeforeAnyEntryTheTurnIsPlainWithNoSince(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, finale.Add(-time.Minute))

	assert.True(t, proto.Equal(&gamemasterv1.Turn{Kind: gamemasterv1.EntryKind_ENTRY_KIND_PLAIN}, answer), "%v", answer)
}

func TestTheTimelineIsTheBattlesToComeInThePlayersWords(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)
	save(t, db, schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 2}), Set: "quiet", Kind: schedule.Plain, Description: "Fewer bans.",
		StartsAt: finale, EndsAt: battleEnds, Priority: 2, MadeAt: madeAt,
	})

	timeline := nowAt(t, db, finale.Add(-time.Minute)).GetTimeline()

	assert.True(t, proto.Equal(&gamemasterv1.Timeline{Battles: []*gamemasterv1.Battle{{
		EntryId: rush, StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(battleEnds), Description: "Triple points.",
		Times: 3, EndsSeason: true, RoundId: rush,
	}}, Current: seasonZero, Today: dayLast}, timeline), "season 0 is seeded: %v", timeline)
}

func TestABattleThatStartedLeavesTheTimeline(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	timeline := nowAt(t, db, finale).GetTimeline()

	require.NotNil(t, timeline)
	assert.Empty(t, timeline.GetBattles())
}

func TestTheTimelineCarriesTheRunningSeasonAndTheNextOne(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	require.NoError(t, postgres_season_store.New(db).Save(t.Context(), season.Snapshot{
		Number: 1, Break: time.Hour, Length: 30 * 24 * time.Hour,
	}.Restore()))

	timeline := nowAt(t, db, finale).GetTimeline()
	assert.True(t, proto.Equal(seasonZero, timeline.GetCurrent()), "%v", timeline)
	assert.True(t, proto.Equal(&gamemasterv1.Season{
		Number: 1, StartsAt: timestamppb.New(battleEnds.Add(time.Hour)), EndsAt: timestamppb.New(battleEnds.Add(time.Hour + 30*24*time.Hour)),
		BookId: "season-1",
	}, timeline.GetNext()), "%v", timeline)

	between := nowAt(t, db, battleEnds).GetTimeline()
	assert.Nil(t, between.GetCurrent(), "a season is over at its end")
	assert.Equal(t, uint32(1), between.GetNext().GetNumber())
}

func TestTodayIsTheDayInProgressAndNoneFromTheLastCutoffToTheSeasonsEnd(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)

	assert.True(t, proto.Equal(&gamemasterv1.Day{
		Number: 1, StartsAt: timestamppb.New(season.Origin), EndsAt: timestamppb.New(season.Origin.Add(21 * time.Hour)),
		RoundId: "season-0/day-1",
	}, nowAt(t, db, season.Origin.Add(time.Hour)).GetTimeline().GetToday()))
	assert.True(t, proto.Equal(dayLast, nowAt(t, db, finale.Add(-time.Second)).GetTimeline().GetToday()))
	assert.Nil(t, nowAt(t, db, finale).GetTimeline().GetToday())
}

func TestTheDebriefIsTheLastMomentScoringScoredAndNoneBeforeAny(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)
	store := postgres_moment_store.New(db)

	assert.Nil(t, nowAt(t, db, finale).GetDebrief())

	_, err := store.KeepRound(t.Context(), moments.ScoredRound{Round: "season-0/day-24", Book: "season-0", EndedAt: finale})
	require.NoError(t, err)
	assert.True(t, proto.Equal(&gamemasterv1.Debrief{Moment: &gamemasterv1.Debrief_Day{Day: &gamemasterv1.DayEnded{
		Season: 0, Day: 24, RoundId: "season-0/day-24", EndedAt: timestamppb.New(finale),
	}}}, nowAt(t, db, finale).GetDebrief()))

	_, err = store.KeepRound(t.Context(), moments.ScoredRound{Round: moments.RoundID(rush), Book: "season-0", EndedAt: battleEnds})
	require.NoError(t, err)
	_, err = store.KeepBook(t.Context(), moments.ClosedBook{Book: "season-0", EndedAt: battleEnds})
	require.NoError(t, err)
	assert.True(t, proto.Equal(&gamemasterv1.Debrief{Moment: &gamemasterv1.Debrief_Season{Season: &gamemasterv1.SeasonEnded{
		Season: 0, BookId: "season-0", EndedAt: timestamppb.New(battleEnds), FinaleRoundId: rush,
		FinaleBattle: &gamemasterv1.BattleEnded{
			Season: 0, Number: 1, EntryId: rush, Times: 3, EndsSeason: true, RoundId: rush, EndedAt: timestamppb.New(battleEnds),
		},
	}}}, nowAt(t, db, battleEnds).GetDebrief()), "the book closes after the round that ends with it")
}
