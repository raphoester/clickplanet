package casts_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/screen_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type usernames map[casts.AccountID]string

func (u usernames) Of(context.Context, []casts.AccountID) (map[casts.AccountID]string, error) {
	return map[casts.AccountID]string(u), nil
}

func screening(judge *verdicts.ScriptedJudge) *screen_names_usecase.UseCase {
	return screen_names_usecase.New(verdicts.NewMemoryStore(), judge, cptime.NewFixedClock(at))
}

func TestAPlayerWhoKeepsOutOfTheClipsHasNoNameShown(t *testing.T) {
	ada, bob := account(t, "01"), account(t, "02")
	optOuts := optouts.NewMemoryStore()
	require.NoError(t, optOuts.Keep(t.Context(), bob))
	judge := verdicts.NewScriptedJudge(verdicts.Judgement{Name: "Ada", Shown: true}, verdicts.Judgement{Name: "Bob", Shown: true})

	shown, err := casts.NewShownNames(usernames{ada: "Ada", bob: "Bob"}, optOuts, screening(judge)).
		Shown(t.Context(), []casts.AccountID{ada, bob})
	require.NoError(t, err)

	assert.Equal(t, map[casts.AccountID]string{ada: "Ada"}, shown)
	assert.Equal(t, [][]string{{"Ada"}}, judge.Asked, "a name kept out is not judged")
}

func TestANameThePlatformsForbidIsNotShown(t *testing.T) {
	ada, ira := account(t, "01"), account(t, "02")
	judge := verdicts.NewScriptedJudge(
		verdicts.Judgement{Name: "Ada", Shown: true},
		verdicts.Judgement{Name: "IRA4Ever", Shown: false, Reason: "the IRA, an armed group"},
	)

	shown, err := casts.NewShownNames(usernames{ada: "Ada", ira: "IRA4Ever"}, optouts.NewMemoryStore(), screening(judge)).
		Shown(t.Context(), []casts.AccountID{ada, ira})
	require.NoError(t, err)

	assert.Equal(t, map[casts.AccountID]string{ada: "Ada"}, shown)
}
