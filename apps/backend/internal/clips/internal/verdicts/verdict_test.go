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

func TestAPlayerWhoRenamesIsJudgedAgain(t *testing.T) {
	kept := judged(map[verdicts.AccountID]string{ira: "Ada2"}, verdicts.Judgement{Name: "Ada2", Shown: true})

	shown, unjudged := verdicts.Screen(map[verdicts.AccountID]string{ira: "IRA4Ever"}, kept)

	assert.Empty(t, shown)
	assert.Equal(t, map[verdicts.AccountID]string{ira: "IRA4Ever"}, unjudged)
}

func TestAShownNameKeepsNoReason(t *testing.T) {
	kept := judged(map[verdicts.AccountID]string{ada: "Ada"}, verdicts.Judgement{Name: "Ada", Shown: true, Reason: "fine"})

	assert.Equal(t, verdicts.Snapshot{Account: ada, Name: "Ada", Shown: true, By: verdicts.ByJudge, At: at}, kept[0].TakeSnapshot())
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
