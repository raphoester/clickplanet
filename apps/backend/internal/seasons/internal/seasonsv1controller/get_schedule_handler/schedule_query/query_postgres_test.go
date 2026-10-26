package schedule_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule/postgres_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_schedule_handler/schedule_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	finale     = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonEnds = finale.Add(2 * time.Hour)
	seasonZero = calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour}}})
	sets       = schedule.PhaseSets{Days: "default", Finale: "rush", Over: "default"}
	calm       = schedule.EntryID(uuid.UUID{15: 1})
	frozen     = schedule.EntryID(uuid.UUID{15: 2})
)

func keep(t *testing.T, store *postgres_entry_store.Store, entries ...schedule.EntrySnapshot) {
	t.Helper()
	for _, entry := range entries {
		require.NoError(t, store.Save(t.Context(), entry.Restore()))
	}
}

func TestThePhasesAndEveryEntryAreListedWithWhatPlaysNow(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "seasons", migrations.FS)
	madeAt := finale.Add(-time.Hour)
	keep(t, postgres_entry_store.New(db),
		schedule.EntrySnapshot{ID: frozen, Set: "frozen", StartsAt: finale.Add(time.Hour), EndsAt: seasonEnds, Priority: 1, MadeAt: madeAt},
		schedule.EntrySnapshot{ID: calm, Set: "calm", StartsAt: finale.Add(10 * time.Minute), EndsAt: finale.Add(30 * time.Minute), Priority: 2, MadeAt: madeAt},
	)

	answer, err := schedule_query.NewPostgresQuery(db, seasonZero, sets, cptime.NewFixedClock(finale.Add(15*time.Minute))).
		Schedule(t.Context())
	require.NoError(t, err)

	assert.True(t, proto.Equal(&seasonsv1.GetScheduleResponse{
		Phases: []*seasonsv1.Phase{
			{Season: 0, Stage: seasonsv1.Stage_STAGE_DAYS, RuleSet: "default", EndsAt: timestamppb.New(finale)},
			{Season: 0, Stage: seasonsv1.Stage_STAGE_FINALE, RuleSet: "rush", StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(seasonEnds)},
			{Season: 0, Stage: seasonsv1.Stage_STAGE_OVER, RuleSet: "default", StartsAt: timestamppb.New(seasonEnds)},
		},
		Entries: []*seasonsv1.Entry{
			{
				Id: calm.String(), RuleSet: "calm", StartsAt: timestamppb.New(finale.Add(10 * time.Minute)),
				EndsAt: timestamppb.New(finale.Add(30 * time.Minute)), Priority: 2, MadeAt: timestamppb.New(madeAt),
			},
			{
				Id: frozen.String(), RuleSet: "frozen", StartsAt: timestamppb.New(finale.Add(time.Hour)),
				EndsAt: timestamppb.New(seasonEnds), Priority: 1, MadeAt: timestamppb.New(madeAt),
			},
		},
		Playing: &seasonsv1.Playing{RuleSet: "calm", Since: timestamppb.New(finale.Add(10 * time.Minute)), EntryId: calm.String()},
	}, answer), "%v", answer)
}

func TestAPhaseThatPlaysNamesNoEntry(t *testing.T) {
	db := cppg.StartTestServer(t).OpenSchema(t, "seasons", migrations.FS)

	answer, err := schedule_query.NewPostgresQuery(db, seasonZero, sets, cptime.NewFixedClock(finale.Add(time.Minute))).
		Schedule(t.Context())
	require.NoError(t, err)

	assert.True(t, proto.Equal(&seasonsv1.Playing{RuleSet: "rush", Since: timestamppb.New(finale)}, answer.GetPlaying()))
	assert.Empty(t, answer.GetEntries())
}
