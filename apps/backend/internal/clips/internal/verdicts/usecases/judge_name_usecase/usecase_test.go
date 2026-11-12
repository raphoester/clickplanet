package judge_name_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/judge_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/screen_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at       = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	ada, ira = verdicts.AccountID{15: 1}, verdicts.AccountID{15: 2}
)

type usernames struct {
	names map[verdicts.AccountID]string
	err   error
	asked [][]verdicts.AccountID
}

func (u *usernames) Of(_ context.Context, accounts []verdicts.AccountID) (map[verdicts.AccountID]string, error) {
	u.asked = append(u.asked, accounts)
	held := map[verdicts.AccountID]string{}
	for _, account := range accounts {
		if name, named := u.names[account]; named {
			held[account] = name
		}
	}
	return held, u.err
}

func useCaseOf(names *usernames, store *verdicts.MemoryStore, judge *verdicts.ScriptedJudge) *judge_name_usecase.UseCase {
	return judge_name_usecase.New(names, screen_names_usecase.New(store, judge, cptime.NewFixedClock(at)))
}

func TestANameSetIsJudgedAndItsVerdictKept(t *testing.T) {
	store := verdicts.NewMemoryStore()
	judge := verdicts.NewScriptedJudge(verdicts.Judgement{Name: "IRA4Ever", Reason: "the IRA, an armed group"})
	names := &usernames{names: map[verdicts.AccountID]string{ira: "IRA4Ever"}}

	require.NoError(t, useCaseOf(names, store, judge).Execute(t.Context(), ira))

	assert.Equal(t, [][]verdicts.AccountID{{ira}}, names.asked)
	assert.Equal(t, [][]string{{"IRA4Ever"}}, judge.Asked)
	kept, err := store.Verdict(t.Context(), ira)
	require.NoError(t, err)
	assert.Equal(t, verdicts.Snapshot{
		Account: ira, Name: "IRA4Ever", Reason: "the IRA, an armed group", By: verdicts.ByJudge, At: at,
	}, kept.TakeSnapshot())
}

func TestANameJudgedAlreadyAsksNobody(t *testing.T) {
	store := verdicts.NewMemoryStore()
	require.NoError(t, store.Keep(t.Context(), verdicts.Judged(map[verdicts.AccountID]string{ada: "Ada"},
		[]verdicts.Judgement{{Name: "Ada", Shown: true}}, at.Add(-time.Hour))))
	judge := verdicts.NewScriptedJudge()
	judge.FailWith(errors.New("never asked"))

	require.NoError(t, useCaseOf(&usernames{names: map[verdicts.AccountID]string{ada: "Ada"}}, store, judge).
		Execute(t.Context(), ada))

	assert.Empty(t, judge.Asked)
}

func TestARenamedPlayerIsJudgedUnderItsNewName(t *testing.T) {
	store := verdicts.NewMemoryStore()
	require.NoError(t, store.Keep(t.Context(), verdicts.Judged(map[verdicts.AccountID]string{ada: "Ada"},
		[]verdicts.Judgement{{Name: "Ada", Shown: true}}, at.Add(-time.Hour))))
	judge := verdicts.NewScriptedJudge(verdicts.Judgement{Name: "Ada_L", Shown: true})

	require.NoError(t, useCaseOf(&usernames{names: map[verdicts.AccountID]string{ada: "Ada_L"}}, store, judge).
		Execute(t.Context(), ada))

	assert.Equal(t, [][]string{{"Ada_L"}}, judge.Asked)
	kept, err := store.Verdict(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, verdicts.Snapshot{Account: ada, Name: "Ada_L", Shown: true, By: verdicts.ByJudge, At: at}, kept.TakeSnapshot())
}

func TestAnAccountWithNoUsernameAsksNobody(t *testing.T) {
	store, judge := verdicts.NewMemoryStore(), verdicts.NewScriptedJudge()
	judge.FailWith(errors.New("never asked"))

	require.NoError(t, useCaseOf(&usernames{}, store, judge).Execute(t.Context(), ada))

	assert.Empty(t, judge.Asked)
	_, err := store.Verdict(t.Context(), ada)
	require.ErrorIs(t, err, verdicts.ErrNoVerdict)
}

func TestAJudgeThatFailsKeepsNothingSoTheNameIsAskedAgain(t *testing.T) {
	store, failing := verdicts.NewMemoryStore(), verdicts.NewScriptedJudge()
	failing.FailWith(errors.New("overloaded"))
	names := &usernames{names: map[verdicts.AccountID]string{ada: "Ada"}}

	require.Error(t, useCaseOf(names, store, failing).Execute(t.Context(), ada))
	_, err := store.Verdict(t.Context(), ada)
	require.ErrorIs(t, err, verdicts.ErrNoVerdict)

	answering := verdicts.NewScriptedJudge(verdicts.Judgement{Name: "Ada", Shown: true})
	require.NoError(t, useCaseOf(names, store, answering).Execute(t.Context(), ada))
	assert.Equal(t, [][]string{{"Ada"}}, answering.Asked)
}

func TestAUsernameThatCannotBeReadJudgesNothing(t *testing.T) {
	failure := errors.New("player is down")
	judge := verdicts.NewScriptedJudge()

	err := useCaseOf(&usernames{err: failure}, verdicts.NewMemoryStore(), judge).Execute(t.Context(), ada)

	require.ErrorIs(t, err, failure)
	assert.Empty(t, judge.Asked)
}
