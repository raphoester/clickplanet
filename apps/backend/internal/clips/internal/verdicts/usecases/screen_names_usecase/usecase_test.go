package screen_names_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/screen_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at            = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	ada, ira, bob = verdicts.AccountID{15: 1}, verdicts.AccountID{15: 2}, verdicts.AccountID{15: 3}
	adaShown      = verdicts.Judgement{Name: "Ada", Shown: true}
	iraHidden     = verdicts.Judgement{Name: "IRA4Ever", Shown: false, Reason: "the IRA, an armed group"}
)

func TestEachNameIsJudgedOnceAndItsVerdictKept(t *testing.T) {
	store, judge := verdicts.NewMemoryStore(), verdicts.NewScriptedJudge(adaShown, iraHidden)
	useCase := screen_names_usecase.New(store, judge, cptime.NewFixedClock(at))
	names := map[verdicts.AccountID]string{ada: "Ada", ira: "IRA4Ever"}

	first, err := useCase.Execute(t.Context(), names)
	require.NoError(t, err)
	again, err := useCase.Execute(t.Context(), names)
	require.NoError(t, err)

	assert.Equal(t, map[verdicts.AccountID]string{ada: "Ada"}, first)
	assert.Equal(t, first, again)
	assert.Equal(t, [][]string{{"Ada", "IRA4Ever"}}, judge.Asked, "judged once, the names in order")
	kept, err := store.Verdict(t.Context(), ira)
	require.NoError(t, err)
	assert.Equal(t, verdicts.Snapshot{
		Account: ira, Name: "IRA4Ever", Reason: "the IRA, an armed group", By: verdicts.ByJudge, At: at,
	}, kept.TakeSnapshot())
}

func TestOnlyTheNewNamesAreJudged(t *testing.T) {
	store, judge := verdicts.NewMemoryStore(), verdicts.NewScriptedJudge(adaShown, iraHidden)
	useCase := screen_names_usecase.New(store, judge, cptime.NewFixedClock(at))
	_, err := useCase.Execute(t.Context(), map[verdicts.AccountID]string{ada: "Ada"})
	require.NoError(t, err)

	shown, err := useCase.Execute(t.Context(), map[verdicts.AccountID]string{ada: "Ada", ira: "IRA4Ever"})
	require.NoError(t, err)

	assert.Equal(t, map[verdicts.AccountID]string{ada: "Ada"}, shown)
	assert.Equal(t, [][]string{{"Ada"}, {"IRA4Ever"}}, judge.Asked)
}

func TestANameTheJudgeLeftOutIsHiddenAndAskedAgain(t *testing.T) {
	judge := verdicts.NewScriptedJudge(adaShown)
	useCase := screen_names_usecase.New(verdicts.NewMemoryStore(), judge, cptime.NewFixedClock(at))
	names := map[verdicts.AccountID]string{ada: "Ada", bob: "Bob"}

	shown, err := useCase.Execute(t.Context(), names)
	require.NoError(t, err)
	_, err = useCase.Execute(t.Context(), names)
	require.NoError(t, err)

	assert.Equal(t, map[verdicts.AccountID]string{ada: "Ada"}, shown)
	assert.Equal(t, [][]string{{"Ada", "Bob"}, {"Bob"}}, judge.Asked)
}

func TestAJudgeThatFailsShowsNoNewNameAndKeepsNothing(t *testing.T) {
	store, judge := verdicts.NewMemoryStore(), verdicts.NewScriptedJudge(adaShown)
	judge.FailWith(errors.New("overloaded"))

	_, err := screen_names_usecase.New(store, judge, cptime.NewFixedClock(at)).
		Execute(t.Context(), map[verdicts.AccountID]string{ada: "Ada"})

	require.Error(t, err)
	_, err = store.Verdict(t.Context(), ada)
	require.ErrorIs(t, err, verdicts.ErrNoVerdict)
}

func TestNamesAllJudgedAskNobody(t *testing.T) {
	judge := verdicts.NewScriptedJudge()
	judge.FailWith(errors.New("never asked"))

	shown, err := screen_names_usecase.New(verdicts.NewMemoryStore(), judge, cptime.NewFixedClock(at)).
		Execute(t.Context(), map[verdicts.AccountID]string{})

	require.NoError(t, err)
	assert.Empty(t, shown)
	assert.Empty(t, judge.Asked)
}
