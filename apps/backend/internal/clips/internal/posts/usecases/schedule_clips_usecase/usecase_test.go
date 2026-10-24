package schedule_clips_usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/schedule_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 10, 12, 12, 40, 0, 0, time.UTC)
	plan   = posts.Plan{Review: 4 * time.Hour, Spacing: 2 * time.Hour, Platforms: []posts.Platform{posts.TikTok, posts.YouTube}}
	first  = now.Add(4 * time.Hour)
	second = now.Add(6 * time.Hour)
	length = posts.Experiments{{Name: "length", Arms: posts.EvenArms("as-chosen", "short"), Control: "as-chosen"}}
)

type folder struct {
	clips []posts.Clip
	err   error
}

func (f folder) Rendered(context.Context) ([]posts.Clip, error) {
	return f.clips, f.err
}

func key(title string) posts.StoryKey {
	return posts.StoryKey{Flag: "dz", Continent: title}
}

func clip(t *testing.T, pick posts.Pick, title string) posts.Clip {
	t.Helper()

	return clipIn(t, pick, title, nil)
}

func clipIn(t *testing.T, pick posts.Pick, title string, variants posts.Variants) posts.Clip {
	t.Helper()

	restored, err := posts.ClipSnapshot{
		Pick: pick, Title: title, Teasers: []string{title + " 1", title + " 2"}, Key: key(title), Video: "clip.mp4", Variants: variants,
	}.Restore()
	require.NoError(t, err)
	return restored
}

type run struct {
	service *posts.ScriptedService
	usecase *schedule_clips_usecase.UseCase
}

func scheduling(clips folder, refusals ...error) run {
	clock := cptime.NewFixedClock(now)
	service := posts.NewScriptedService(refusals...)
	sender := posts.NewSender(service, posts.Backoff{time.Minute}, posts.NewRecordedPause(clock), clock)
	return run{service: service, usecase: schedule_clips_usecase.New(clips, plan, clock, posts.LoadedDice(1), sender, length)}
}

func on(failures ...string) []posts.PlatformOutcome {
	return []posts.PlatformOutcome{{Platform: posts.TikTok, Failure: failures[0]}, {Platform: posts.YouTube, Failure: failures[1]}}
}

func TestEachRenderedClipIsScheduledBestFirstAndTheReportSaysSo(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 2, "SECOND"), clip(t, 1, "BEST")}})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first, Post: "post-BEST", Platforms: on("", "")},
		{Title: "SECOND", At: second, Post: "post-SECOND", Platforms: on("", "")},
	}, report)
	assert.True(t, report.Complete())
	require.Len(t, r.service.Scheduled, 2)
	assert.Equal(t, []string{"BEST", "SECOND"}, []string{r.service.Scheduled[0].Title, r.service.Scheduled[1].Title})
	assert.Equal(t, []time.Time{first, second}, []time.Time{r.service.Scheduled[0].At, r.service.Scheduled[1].At})
	assert.True(t, strings.HasPrefix(r.service.Scheduled[0].TikTok.Caption, "BEST 2 | CLICKPLANET\n"))
	assert.Equal(t, "BEST 2 | CLICKPLANET", r.service.Scheduled[0].YouTube.Title)
	assert.Equal(t, []posts.Platform{posts.TikTok, posts.YouTube}, r.service.Scheduled[0].Platforms)
	assert.Equal(t, "browser", r.service.Scheduled[0].CallToAction)
}

func TestAClipTheServiceRefusesIsInTheReportAndTheOthersStillGoOut(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}}, errors.New("the caption is too long"))

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first, Platforms: on("the caption is too long", "the caption is too long")},
		{Title: "SECOND", At: second, Post: "post-SECOND", Platforms: on("", "")},
	}, report)
	assert.False(t, report.Complete())
}

func TestAPlatformThatRefusesDoesNotStopTheOthers(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}})
	r.service.Refusing = map[posts.Platform]error{posts.YouTube: errors.New("no youtube account")}

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first, Post: "post-BEST", Platforms: on("", "no youtube account")},
		{Title: "SECOND", At: second, Post: "post-SECOND", Platforms: on("", "no youtube account")},
	}, report)
	assert.False(t, report.Complete())
}

func TestARunWithNoRenderedClipFails(t *testing.T) {
	r := scheduling(folder{})

	_, err := r.usecase.Execute(t.Context())

	require.ErrorIs(t, err, schedule_clips_usecase.ErrNoClip)
}

func TestClipsThatCannotBeReadFailTheRun(t *testing.T) {
	unreadable := errors.New("the folder is gone")
	r := scheduling(folder{err: unreadable})

	_, err := r.usecase.Execute(t.Context())

	require.ErrorIs(t, err, unreadable)
	assert.Empty(t, r.service.Scheduled)
}

func TestTheArmsOfEachClipReachItsPostAndTheReport(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clipIn(t, 1, "BEST", posts.Variants{"length": "short"}), clip(t, 2, "SECOND")}})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Variants{"length": "short"}, report[0].Variants)
	assert.Empty(t, report[1].Variants)
	assert.Equal(t, posts.Variants{"length": "short"}, r.service.Scheduled[0].Variants)
}

// The control is what a clip was before the experiment: the same post, word for word, at the same time.
func TestAClipInTheControlArmIsPostedAsAClipInNoExperiment(t *testing.T) {
	control := scheduling(folder{clips: []posts.Clip{clipIn(t, 1, "BEST", posts.Variants{"length": "as-chosen"})}})
	before := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST")}})

	_, err := control.usecase.Execute(t.Context())
	require.NoError(t, err)
	_, err = before.usecase.Execute(t.Context())
	require.NoError(t, err)

	posted := control.service.Scheduled[0]
	posted.Variants = nil
	assert.Equal(t, before.service.Scheduled[0], posted)
}

func TestAClipInAnArmNobodyRunsFailsTheRunBeforeAnyPost(t *testing.T) {
	for name, variants := range map[string]posts.Variants{
		"an experiment that does not run": {"end-card": "podium"},
		"an arm the experiment has not":   {"length": "long"},
	} {
		t.Run(name, func(t *testing.T) {
			r := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST"), clipIn(t, 2, "SECOND", variants)}})

			_, err := r.usecase.Execute(t.Context())

			require.ErrorIs(t, err, posts.ErrUnknownVariant)
			assert.Contains(t, err.Error(), "SECOND")
			assert.Empty(t, r.service.Scheduled)
		})
	}
}
