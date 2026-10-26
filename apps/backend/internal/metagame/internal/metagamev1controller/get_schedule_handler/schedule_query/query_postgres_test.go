package schedule_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_schedule_handler/schedule_query"
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
	rush       = schedule.EntryID(uuid.UUID{15: 1})
	calm       = schedule.EntryID(uuid.UUID{15: 2})
)

func keep(t *testing.T, store *postgres_entry_store.Store, entries ...schedule.EntrySnapshot) {
	t.Helper()
	for _, entry := range entries {
		require.NoError(t, store.Save(t.Context(), entry.Restore()))
	}
}

func TestEveryEntryIsListedByStartWithWhatPlaysNow(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "metagame", migrations.FS)
	keep(t, postgres_entry_store.New(db),
		schedule.EntrySnapshot{
			ID: calm, Set: "calm", Kind: schedule.Plain, StartsAt: finale.Add(10 * time.Minute), EndsAt: finale.Add(30 * time.Minute),
			Priority: 2, MadeAt: madeAt,
		},
		schedule.EntrySnapshot{ID: rush, Set: "rush", Kind: schedule.Battle, StartsAt: finale, EndsAt: battleEnds, Priority: 1, MadeAt: madeAt},
	)

	answer, err := schedule_query.NewPostgresQuery(db, cptime.NewFixedClock(finale.Add(15*time.Minute))).Schedule(t.Context())
	require.NoError(t, err)

	assert.True(t, proto.Equal(&metagamev1.GetScheduleResponse{
		Entries: []*metagamev1.Entry{
			{
				Id: rush.String(), RuleSet: "rush", Kind: metagamev1.EntryKind_ENTRY_KIND_BATTLE, StartsAt: timestamppb.New(finale),
				EndsAt: timestamppb.New(battleEnds), Priority: 1, MadeAt: timestamppb.New(madeAt),
			},
			{
				Id: calm.String(), RuleSet: "calm", Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN,
				StartsAt: timestamppb.New(finale.Add(10 * time.Minute)), EndsAt: timestamppb.New(finale.Add(30 * time.Minute)),
				Priority: 2, MadeAt: timestamppb.New(madeAt),
			},
		},
		Playing: &metagamev1.Playing{
			RuleSet: "calm", Since: timestamppb.New(finale.Add(10 * time.Minute)), EntryId: calm.String(),
			Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN,
		},
	}, answer), "%v", answer)
}

func TestWithNoEntryRunningDefaultPlaysAndNamesNone(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "metagame", migrations.FS)
	keep(t, postgres_entry_store.New(db),
		schedule.EntrySnapshot{ID: rush, Set: "rush", Kind: schedule.Battle, StartsAt: finale, EndsAt: battleEnds, Priority: 1, MadeAt: madeAt})

	answer, err := schedule_query.NewPostgresQuery(db, cptime.NewFixedClock(battleEnds.Add(time.Minute))).Schedule(t.Context())
	require.NoError(t, err)

	assert.True(t, proto.Equal(&metagamev1.Playing{
		RuleSet: "default", Since: timestamppb.New(battleEnds), Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN,
	}, answer.GetPlaying()))
}
