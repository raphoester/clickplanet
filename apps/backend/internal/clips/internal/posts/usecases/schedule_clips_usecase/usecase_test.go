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
	now     = time.Date(2026, 10, 12, 12, 40, 0, 0, time.UTC)
	both    = []posts.Platform{posts.TikTok, posts.YouTube}
	youtube = []posts.Platform{posts.YouTube}
	plan    = posts.Plan{Platforms: both}
	// As the job runs: TikTok for a run's best clip alone.
	asap   = posts.Plan{Platforms: both, BestOnly: []posts.Platform{posts.TikTok}}
	first  = posts.Slot{At: now.Add(4 * time.Hour), Platforms: both}
	second = posts.Slot{At: now.Add(6 * time.Hour), Platforms: both}
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

	restored, err := posts.ClipSnapshot{
		Pick: pick, Title: title, Teasers: []string{title + " 1", title + " 2"}, Key: key(title), Video: "clip.mp4", Seconds: 10,
	}.Restore()
	require.NoError(t, err)
	return restored
}

type run struct {
	calendar *posts.ScriptedPlanner
	service  *posts.ScriptedService
	usecase  *schedule_clips_usecase.UseCase
}

// The calendar gives the first slot, then the second.
func scheduling(clips folder, refusals ...error) run {
	return schedulingWith(plan, &posts.ScriptedPlanner{Slots: []posts.Slot{first, second}}, clips, refusals...)
}

func schedulingWith(p posts.Plan, calendar *posts.ScriptedPlanner, clips folder, refusals ...error) run {
	clock := cptime.NewFixedClock(now)
	service := posts.NewScriptedService(refusals...)
	sender := posts.NewSender(calendar, service, posts.Backoff{15 * time.Minute}, posts.NewRecordedPause(clock))
	return run{calendar: calendar, service: service, usecase: schedule_clips_usecase.New(clips, p, posts.LoadedDice(1), sender)}
}

func on(failures ...string) []posts.PlatformOutcome {
	return []posts.PlatformOutcome{{Platform: posts.TikTok, Failure: failures[0]}, {Platform: posts.YouTube, Failure: failures[1]}}
}

func TestEachRenderedClipIsScheduledBestFirstAtTheSlotTheCalendarGivesAndTheReportSaysSo(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 2, "SECOND"), clip(t, 1, "BEST")}})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first.At, Post: "post-BEST", Platforms: on("", "")},
		{Title: "SECOND", At: second.At, Post: "post-SECOND", Platforms: on("", "")},
	}, report)
	assert.True(t, report.Complete())
	require.Len(t, r.service.Scheduled, 2)
	assert.Equal(t, []string{"BEST", "SECOND"}, []string{r.service.Scheduled[0].Title, r.service.Scheduled[1].Title})
	assert.Equal(t, []time.Time{first.At, second.At}, []time.Time{r.service.Scheduled[0].At, r.service.Scheduled[1].At})
	assert.True(t, strings.HasPrefix(r.service.Scheduled[0].TikTok.Caption, "BEST 2 | CLICKPLANET\n"))
	assert.Equal(t, "BEST 2 | CLICKPLANET", r.service.Scheduled[0].YouTube.Title)
	assert.Equal(t, both, r.service.Scheduled[0].Platforms)
	assert.Equal(t, "browser", r.service.Scheduled[0].CallToAction)
	assert.Equal(t, [2]int{1, 2}, [2]int{r.service.Scheduled[0].RunPosition, r.service.Scheduled[0].RunSize})
}

func TestEachClipAsksTheCalendarForThePlatformsOfItsPlace(t *testing.T) {
	r := schedulingWith(asap, &posts.ScriptedPlanner{Now: now}, folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}})

	_, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, [][]posts.Platform{both, youtube}, r.calendar.Asked, "TikTok takes the best alone")
}

func TestAClipTheServiceRefusesIsInTheReportAndTheOthersStillGoOut(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}}, errors.New("the caption is too long"))

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first.At, Platforms: on("the caption is too long", "the caption is too long")},
		{Title: "SECOND", At: second.At, Post: "post-SECOND", Platforms: on("", "")},
	}, report)
	assert.False(t, report.Complete())
}

func TestAPlatformThatRefusesDoesNotStopTheOthers(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}})
	r.service.Refusing = map[posts.Platform]error{posts.YouTube: errors.New("no youtube account")}

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first.At, Post: "post-BEST", Platforms: on("", "no youtube account")},
		{Title: "SECOND", At: second.At, Post: "post-SECOND", Platforms: on("", "no youtube account")},
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

func TestAClipNoPlatformTakesIsHeldAndTheRunIsStillComplete(t *testing.T) {
	r := schedulingWith(asap, &posts.ScriptedPlanner{Held: true}, folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, r.service.Scheduled)
	assert.Equal(t, posts.Report{{Title: "BEST", Held: true}, {Title: "SECOND", Held: true}}, report)
	assert.True(t, report.Complete())
}

func TestACalendarThatCannotBeReadFailsEachClipOnThePlatformsItAsked(t *testing.T) {
	unreachable := errors.New("ssh failed to ask PlanPost: Permission denied (publickey)")
	r := schedulingWith(asap, &posts.ScriptedPlanner{Failed: unreachable}, folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, r.service.Scheduled)
	assert.Equal(t, on(unreachable.Error(), unreachable.Error()), report[0].Platforms)
	assert.Equal(t, []posts.PlatformOutcome{{Platform: posts.YouTube, Failure: unreachable.Error()}}, report[1].Platforms)
	assert.False(t, report.Complete())
}

func TestAPostAWaitDelayedGoesAtTheSlotPlannedAfterTheWait(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST")}}, &posts.BusyError{RetryAfter: 10 * time.Minute, Reason: "rate limit"})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, second.At, report[0].At)
	assert.Len(t, r.calendar.Asked, 2)
}
