package set_verdict_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/set_verdict_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at  = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	ira = verdicts.AccountID{15: 2}
)

func TestTheOperatorShowsANameTheJudgeHid(t *testing.T) {
	store := verdicts.NewMemoryStore()
	require.NoError(t, store.Keep(t.Context(), verdicts.Judged(map[verdicts.AccountID]string{ira: "Irabella"},
		[]verdicts.Judgement{{Name: "Irabella", Reason: "the IRA"}}, at)))

	_, err := set_verdict_usecase.New(store, cptime.NewFixedClock(at.Add(time.Hour))).
		Execute(t.Context(), ira, true, "a first name")
	require.NoError(t, err)

	kept, err := store.Verdict(t.Context(), ira)
	require.NoError(t, err)
	assert.Equal(t, verdicts.Snapshot{
		Account: ira, Name: "Irabella", Shown: true, By: verdicts.ByOperator, At: at.Add(time.Hour),
	}, kept.TakeSnapshot())
}

func TestANameNeverJudgedHasNoVerdictToSet(t *testing.T) {
	_, err := set_verdict_usecase.New(verdicts.NewMemoryStore(), cptime.NewFixedClock(at)).
		Execute(t.Context(), ira, false, "the IRA")

	require.ErrorIs(t, err, verdicts.ErrNoVerdict)
}
