package verdicts_query_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/list_name_verdicts_handler/verdicts_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/postgres_verdict_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var at = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

func names(listed *clipsv1.ListNameVerdictsResponse) []string {
	out := make([]string, 0, len(listed.GetVerdicts()))
	for _, verdict := range listed.GetVerdicts() {
		out = append(out, verdict.GetName())
	}
	return out
}

func TestTheVerdictsAreListedTheNewestFirstAndTheHiddenAlone(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "clips", migrations.FS)
	store, query := postgres_verdict_store.New(db), verdicts_query.NewPostgresQuery(db)
	ada, ira := verdicts.AccountID(uuid.New()), verdicts.AccountID(uuid.New())
	require.NoError(t, store.Keep(t.Context(), verdicts.Judged(map[verdicts.AccountID]string{ada: "Ada"},
		[]verdicts.Judgement{{Name: "Ada", Shown: true}}, at)))
	require.NoError(t, store.Keep(t.Context(), verdicts.Judged(map[verdicts.AccountID]string{ira: "IRA4Ever"},
		[]verdicts.Judgement{{Name: "IRA4Ever", Reason: "the IRA, an armed group"}}, at.Add(time.Minute))))

	every, err := query.Verdicts(t.Context(), false)
	require.NoError(t, err)
	hidden, err := query.Verdicts(t.Context(), true)
	require.NoError(t, err)

	assert.Equal(t, []string{"IRA4Ever", "Ada"}, names(every))
	assert.Equal(t, []string{"IRA4Ever"}, names(hidden))
	assert.Equal(t, ira.String(), hidden.GetVerdicts()[0].GetAccountId())
	assert.Equal(t, "the IRA, an armed group", hidden.GetVerdicts()[0].GetReason())
	assert.Equal(t, clipsv1.VerdictBy_VERDICT_BY_JUDGE, hidden.GetVerdicts()[0].GetDecidedBy())
	assert.Equal(t, at.Add(time.Minute), hidden.GetVerdicts()[0].GetDecidedAt().AsTime())
}
