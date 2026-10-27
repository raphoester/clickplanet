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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/postgres_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	finale     = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	battleEnds = finale.Add(2 * time.Hour)
	madeAt     = finale.Add(-time.Hour)
)

func battle(t *testing.T, db cppg.Conn) {
	t.Helper()
	save(t, db, schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 1}), Set: "rush", Kind: schedule.Battle, Description: "Triple points.",
		StartsAt: finale, EndsAt: battleEnds, Priority: 1, MadeAt: madeAt,
	})
}

func save(t *testing.T, db cppg.Conn, entry schedule.EntrySnapshot) {
	t.Helper()
	require.NoError(t, postgres_entry_store.New(db).Save(t.Context(), entry.Restore()))
}

func nowAt(t *testing.T, db cppg.Conn, at time.Time) *gamemasterv1.GetTurnResponse {
	t.Helper()
	answer, err := turn_query.NewPostgresQuery(db, cptime.NewFixedClock(at)).Now(t.Context())
	require.NoError(t, err)
	return answer
}

func turnAt(t *testing.T, db cppg.Conn, at time.Time) *gamemasterv1.Turn {
	t.Helper()
	return nowAt(t, db, at).GetTurn()
}

func TestWhileAnEntryRunsTheTurnIsItsIdItsKindItsWordsAndItsEnd(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, finale.Add(time.Minute))

	assert.True(t, proto.Equal(&gamemasterv1.Turn{
		Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, Description: "Triple points.", Since: timestamppb.New(finale),
		EndsAt: timestamppb.New(battleEnds), EntryId: uuid.UUID{15: 1}.String(),
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
		EntryId: uuid.UUID{15: 1}.String(), StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(battleEnds),
		Description: "Triple points.",
	}}}, timeline), "%v", timeline)
}

func TestABattleThatStartedLeavesTheTimeline(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "gamemaster", migrations.FS)
	battle(t, db)

	timeline := nowAt(t, db, finale).GetTimeline()

	require.NotNil(t, timeline)
	assert.Empty(t, timeline.GetBattles())
}
