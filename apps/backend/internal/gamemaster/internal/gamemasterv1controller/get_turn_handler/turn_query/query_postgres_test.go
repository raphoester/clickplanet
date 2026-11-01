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
		BookId: "season-0", FinaleRoundId: "season-0/finale",
	}
	early     = uuid.UUID{15: 1}.String()
	earlyAt   = finale.Add(-27 * time.Hour)
	earlyEnds = earlyAt.Add(2 * time.Hour)
	dayLast   = &gamemasterv1.Day{
		Number: 24, StartsAt: timestamppb.New(finale.Add(-24 * time.Hour)), EndsAt: timestamppb.New(finale), RoundId: "season-0/day-24",
	}
)

// Season 0 is seeded with its finale, rush × 3 from 21:00 to 23:00 on 31 Oct; a battle entry runs the day before.
func battle(t *testing.T, db cppg.Conn) {
	t.Helper()
	save(t, db, schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 1}), Set: "storm", Kind: schedule.Battle, Description: "Double points.",
		StartsAt: earlyAt, EndsAt: earlyEnds, Priority: 1, Times: 2, MadeAt: madeAt,
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

	answer := turnAt(t, db, earlyAt.Add(time.Minute))

	assert.True(t, proto.Equal(&gamemasterv1.Turn{
		Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, Description: "Double points.", Since: timestamppb.New(earlyAt),
		EndsAt: timestamppb.New(earlyEnds), EntryId: early, RoundId: early, Times: 2,
	}, answer), "%v", answer)
}

func TestWhileTheFinalePlaysTheTurnIsTheSeasonsAboveAnyEntry(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	save(t, db, schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 2}), Set: "quiet", Kind: schedule.Plain, Description: "Fewer bans.",
		StartsAt: finale.Add(-time.Hour), EndsAt: battleEnds.Add(time.Hour), Priority: 9, MadeAt: madeAt,
	})

	answer := turnAt(t, db, finale.Add(time.Minute))

	assert.True(t, proto.Equal(&gamemasterv1.Turn{
		Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, Description: "Triple points.", Since: timestamppb.New(finale),
		EndsAt: timestamppb.New(battleEnds), RoundId: "season-0/finale", Times: 3, EndsSeason: true,
	}, answer), "%v", answer)
}

func TestWithNoEntryRunningTheTurnIsPlainSinceTheLastEnd(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, earlyEnds)

	assert.True(t, proto.Equal(&gamemasterv1.Turn{
		Kind: gamemasterv1.EntryKind_ENTRY_KIND_PLAIN, Since: timestamppb.New(earlyEnds),
	}, answer), "%v", answer)
}

func TestBeforeAnyEntryTheTurnIsPlainWithNoSince(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, earlyAt.Add(-time.Minute))

	assert.True(t, proto.Equal(&gamemasterv1.Turn{Kind: gamemasterv1.EntryKind_ENTRY_KIND_PLAIN}, answer), "%v", answer)
}

func TestTheTimelineIsTheBattlesToComeInThePlayersWordsTheFinaleAmongThem(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)
	save(t, db, schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 2}), Set: "quiet", Kind: schedule.Plain, Description: "Fewer bans.",
		StartsAt: earlyAt, EndsAt: earlyEnds, Priority: 2, MadeAt: madeAt,
	})

	timeline := nowAt(t, db, earlyAt.Add(-time.Minute)).GetTimeline()

	assert.True(t, proto.Equal(&gamemasterv1.Timeline{Battles: []*gamemasterv1.Battle{{
		EntryId: early, StartsAt: timestamppb.New(earlyAt), EndsAt: timestamppb.New(earlyEnds), Description: "Double points.",
		Times: 2, RoundId: early,
	}, {
		StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(battleEnds), Description: "Triple points.", Times: 3,
		EndsSeason: true, RoundId: "season-0/finale",
	}}, Current: seasonZero, Today: &gamemasterv1.Day{
		Number: 23, StartsAt: timestamppb.New(finale.Add(-48 * time.Hour)), EndsAt: timestamppb.New(finale.Add(-24 * time.Hour)),
		RoundId: "season-0/day-23",
	}}, timeline), "season 0 and its finale are seeded: %v", timeline)
}

func TestABattleThatStartedLeavesTheTimeline(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	assert.Len(t, nowAt(t, db, earlyAt).GetTimeline().GetBattles(), 1, "the finale is still to come")
	assert.Empty(t, nowAt(t, db, finale).GetTimeline().GetBattles())
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
	assert.Empty(t, timeline.GetEnded())

	between := nowAt(t, db, battleEnds).GetTimeline()
	assert.Nil(t, between.GetCurrent(), "a season is over at its end")
	assert.Equal(t, uint32(1), between.GetNext().GetNumber())
	require.Len(t, between.GetEnded(), 1)
	assert.True(t, proto.Equal(seasonZero, between.GetEnded()[0]), "%v", between)
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

	_, err = store.KeepRound(t.Context(), moments.ScoredRound{Round: "season-0/finale", Book: "season-0", EndedAt: battleEnds})
	require.NoError(t, err)
	finaleEnded := &gamemasterv1.BattleEnded{
		Season: 0, Number: 2, Times: 3, EndsSeason: true, RoundId: "season-0/finale", EndedAt: timestamppb.New(battleEnds),
	}
	assert.True(t, proto.Equal(&gamemasterv1.Debrief{Moment: &gamemasterv1.Debrief_Battle{Battle: finaleEnded}},
		nowAt(t, db, battleEnds).GetDebrief()))
	_, err = store.KeepBook(t.Context(), moments.ClosedBook{Book: "season-0", EndedAt: battleEnds})
	require.NoError(t, err)
	assert.True(t, proto.Equal(&gamemasterv1.Debrief{Moment: &gamemasterv1.Debrief_Season{Season: &gamemasterv1.SeasonEnded{
		Season: 0, BookId: "season-0", EndedAt: timestamppb.New(battleEnds), FinaleRoundId: "season-0/finale",
		FinaleBattle: finaleEnded,
	}}}, nowAt(t, db, battleEnds).GetDebrief()), "the book closes after the round that ends with it")
}
