package postgres_verdict_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/postgres_verdict_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

var at = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

func TestAVerdictIsKeptReplacedAndForgotten(t *testing.T) {
	store := postgres_verdict_store.New(cppg.StartTestServer(t).OpenSchema(t, "clips", migrations.FS))
	ada, ira, bob := verdicts.AccountID(uuid.New()), verdicts.AccountID(uuid.New()), verdicts.AccountID(uuid.New())
	names := map[verdicts.AccountID]string{ada: "Ada", ira: "IRA4Ever"}
	judged := verdicts.Judged(names, []verdicts.Judgement{
		{Name: "Ada", Shown: true},
		{Name: "IRA4Ever", Reason: "the IRA, an armed group"},
	}, at)

	require.NoError(t, store.Keep(t.Context(), judged))
	found, err := store.Of(t.Context(), []verdicts.AccountID{ada, ira, bob})
	require.NoError(t, err)
	assert.ElementsMatch(t, judged, found)

	hidden, err := store.Verdict(t.Context(), ira)
	require.NoError(t, err)
	shown := hidden.Overruled(true, "", at.Add(time.Hour))
	require.NoError(t, store.Keep(t.Context(), []verdicts.Verdict{shown}))
	replaced, err := store.Verdict(t.Context(), ira)
	require.NoError(t, err)
	assert.Equal(t, shown.TakeSnapshot(), replaced.TakeSnapshot())

	require.NoError(t, store.Forget(t.Context(), ira))
	_, err = store.Verdict(t.Context(), ira)
	require.ErrorIs(t, err, verdicts.ErrNoVerdict)
}
