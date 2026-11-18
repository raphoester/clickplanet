package verdicts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
)

var (
	at  = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	ada = verdicts.AccountID{15: 1}
)

func judged(names map[verdicts.AccountID]string, judgements ...verdicts.Judgement) []verdicts.Verdict {
	return verdicts.Judged(names, judgements, at)
}

func TestAShownNameKeepsNoReason(t *testing.T) {
	kept := judged(map[verdicts.AccountID]string{ada: "Ada"}, verdicts.Judgement{Name: "Ada", Shown: true, Reason: "fine"})

	assert.Equal(t, verdicts.Snapshot{Account: ada, Name: "Ada", Shown: true, By: verdicts.ByJudge, At: at}, kept[0].TakeSnapshot())
}
