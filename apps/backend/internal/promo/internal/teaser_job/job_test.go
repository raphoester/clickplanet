package teaser_job_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
)

func TestAFolderIsNamedByItsBattleWithNoMarkAPathReads(t *testing.T) {
	for id, name := range map[teaser_job.BattleID]string{
		"0b8e5c2a-3f2e-4c1a-9a51-7f2d3c4b5a69": "0b8e5c2a-3f2e-4c1a-9a51-7f2d3c4b5a69",
		"season-1/finale":                      "season-1_finale",
		"../../etc":                            ".._.._etc",
		"..":                                   "_..",
		"":                                     "_",
	} {
		assert.Equal(t, name, teaser_job.Due{Battle: id}.FolderName(), id)
	}
}

func TestAReportFailsOnAFailureOrAPostEveryPlatformRefused(t *testing.T) {
	posted := teaser_job.Outcome{Result: teaser_job.Posted, Platforms: []string{"tiktok", "youtube"}, Refused: map[string]string{"youtube": "no account"}}

	assert.False(t, teaser_job.Report{posted, {Result: teaser_job.Held}, {Result: teaser_job.Skipped}}.Failed())
	assert.True(t, teaser_job.Report{posted, {Result: teaser_job.Failed}}.Failed())
	everywhere := posted
	everywhere.Refused = map[string]string{"tiktok": "no account", "youtube": "no account"}
	assert.True(t, teaser_job.Report{everywhere}.Failed())
}

func TestABackoffWaitsItsStepsOrWhatWasAskedAndNoMore(t *testing.T) {
	backoff := teaser_job.Backoff{time.Minute, 5 * time.Minute, 15 * time.Minute}

	wait, again := backoff.Wait(1, 0)
	assert.True(t, again)
	assert.Equal(t, time.Minute, wait)
	wait, _ = backoff.Wait(2, 7*time.Minute)
	assert.Equal(t, 7*time.Minute, wait)
	_, again = backoff.Wait(4, 0)
	assert.False(t, again)
	_, again = backoff.Wait(1, time.Hour)
	assert.False(t, again, "a wait longer than the backoff's last is not waited")
}
