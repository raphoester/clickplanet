package verdicts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
)

var (
	at       = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	ada, ira = verdicts.AccountID{15: 1}, verdicts.AccountID{15: 2}
)

func judged(names map[verdicts.AccountID]string, judgements ...verdicts.Judgement) []verdicts.Verdict {
	return verdicts.Judged(names, judgements, at)
}

func TestAJudgedNameIsShownOrHidden(t *testing.T) {
	names := map[verdicts.AccountID]string{ada: "Ada", ira: "IRA4Ever"}
	kept := judged(names,
		verdicts.Judgement{Name: "Ada", Shown: true},
		verdicts.Judgement{Name: "IRA4Ever", Shown: false, Reason: "the IRA, an armed group"})

	shown, unjudged := verdicts.Screen(names, kept)

	assert.Equal(t, map[verdicts.AccountID]string{ada: "Ada"}, shown)
	assert.Empty(t, unjudged)
}

func TestANameWithNoVerdictIsNotShownButJudged(t *testing.T) {
	shown, unjudged := verdicts.Screen(map[verdicts.AccountID]string{ada: "Ada"}, nil)

	assert.Empty(t, shown)
	assert.Equal(t, map[verdicts.AccountID]string{ada: "Ada"}, unjudged)
}

func TestAPlayerWhoRenamesIsJudgedAgain(t *testing.T) {
	kept := judged(map[verdicts.AccountID]string{ira: "Ada2"}, verdicts.Judgement{Name: "Ada2", Shown: true})

	shown, unjudged := verdicts.Screen(map[verdicts.AccountID]string{ira: "IRA4Ever"}, kept)

	assert.Empty(t, shown)
	assert.Equal(t, map[verdicts.AccountID]string{ira: "IRA4Ever"}, unjudged)
}

func TestANameTheJudgeLeftOutHasNoVerdict(t *testing.T) {
	kept := judged(map[verdicts.AccountID]string{ada: "Ada", ira: "IRA4Ever"}, verdicts.Judgement{Name: "Ada", Shown: true})

	require.Len(t, kept, 1)
	assert.Equal(t, ada, kept[0].TakeSnapshot().Account)
}

func TestAShownNameKeepsNoReason(t *testing.T) {
	kept := judged(map[verdicts.AccountID]string{ada: "Ada"}, verdicts.Judgement{Name: "Ada", Shown: true, Reason: "fine"})

	assert.Equal(t, verdicts.Snapshot{Account: ada, Name: "Ada", Shown: true, By: verdicts.ByJudge, At: at}, kept[0].TakeSnapshot())
}

func TestTheOperatorOverrulesTheJudgeOnTheSameName(t *testing.T) {
	kept := judged(map[verdicts.AccountID]string{ira: "IRA4Ever"}, verdicts.Judgement{Name: "IRA4Ever", Reason: "the IRA"})

	overruled := kept[0].Overruled(true, "a joke", at.Add(time.Hour))

	assert.Equal(t, verdicts.Snapshot{
		Account: ira, Name: "IRA4Ever", Shown: true, By: verdicts.ByOperator, At: at.Add(time.Hour),
	}, overruled.TakeSnapshot())
}

func TestADeciderIsJudgeOrOperator(t *testing.T) {
	for _, value := range []string{"judge", "operator"} {
		decider, err := verdicts.DeciderOf(value)
		require.NoError(t, err)
		assert.Equal(t, verdicts.Decider(value), decider)
	}
	_, err := verdicts.DeciderOf("audit")
	require.ErrorIs(t, err, verdicts.ErrUnknownDecider)
}
