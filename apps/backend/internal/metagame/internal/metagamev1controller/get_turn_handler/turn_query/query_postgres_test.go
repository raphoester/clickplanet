package turn_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_turn_handler/turn_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule/postgres_entry_store"
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
	require.NoError(t, postgres_entry_store.New(db).Save(t.Context(), schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 1}), Set: "rush", Kind: schedule.Battle, StartsAt: finale, EndsAt: battleEnds,
		Priority: 1, MadeAt: madeAt,
	}.Restore()))
}

func turnAt(t *testing.T, db cppg.Conn, at time.Time) *metagamev1.GetTurnResponse {
	t.Helper()
	answer, err := turn_query.NewPostgresQuery(db, cptime.NewFixedClock(at)).Turn(t.Context())
	require.NoError(t, err)
	return answer
}

func TestWhileAnEntryRunsTheTurnIsItsKindAndItsEnd(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "metagame", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, finale.Add(time.Minute))

	assert.True(t, proto.Equal(&metagamev1.GetTurnResponse{
		Kind: metagamev1.EntryKind_ENTRY_KIND_BATTLE, Since: timestamppb.New(finale), EndsAt: timestamppb.New(battleEnds),
	}, answer), "%v", answer)
}

func TestWithNoEntryRunningTheTurnIsPlainSinceTheLastEnd(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "metagame", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, battleEnds)

	assert.True(t, proto.Equal(&metagamev1.GetTurnResponse{
		Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN, Since: timestamppb.New(battleEnds),
	}, answer), "%v", answer)
}

func TestBeforeAnyEntryTheTurnIsPlainWithNoSince(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "metagame", migrations.FS)
	battle(t, db)

	answer := turnAt(t, db, finale.Add(-time.Minute))

	assert.True(t, proto.Equal(&metagamev1.GetTurnResponse{Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN}, answer), "%v", answer)
}
