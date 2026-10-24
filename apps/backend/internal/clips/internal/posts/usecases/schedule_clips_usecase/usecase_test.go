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
	budget = posts.Budget{posts.TikTok: 10, posts.YouTube: 8}
	both   = []posts.Platform{posts.TikTok, posts.YouTube}
	plan   = posts.Plan{Review: 4 * time.Hour, Spacing: 2 * time.Hour, Platforms: both, Budget: budget}
	first  = now.Add(4 * time.Hour)
	second = now.Add(6 * time.Hour)
)

type feeds struct {
	feed  posts.Feed
	err   error
	since []time.Time
}

func (f *feeds) Posted(_ context.Context, since time.Time) (posts.Feed, error) {
	f.since = append(f.since, since)
	return f.feed, f.err
}

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

	restored, err := posts.ClipSnapshot{Pick: pick, Title: title, Teasers: []string{title + " 1", title + " 2"}, Key: key(title), Video: "clip.mp4"}.Restore()
	require.NoError(t, err)
	return restored
}

type run struct {
	service *posts.ScriptedService
	feeds   *feeds
	usecase *schedule_clips_usecase.UseCase
}

func scheduling(clips folder, refusals ...error) run {
	return schedulingWith(plan, &feeds{}, clips, refusals...)
}

func schedulingWith(p posts.Plan, posted *feeds, clips folder, refusals ...error) run {
	clock := cptime.NewFixedClock(now)
	service := posts.NewScriptedService(refusals...)
	sender := posts.NewSender(service, posts.Backoff{time.Minute}, posts.NewRecordedPause(clock), clock)
	return run{service: service, feeds: posted, usecase: schedule_clips_usecase.New(clips, posted, p, clock, posts.LoadedDice(1), sender)}
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
	// A clip refused takes no place: the next one goes out in it.
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first, Platforms: on("the caption is too long", "the caption is too long")},
		{Title: "SECOND", At: first, Post: "post-SECOND", Platforms: on("", "")},
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

var asap = posts.Plan{Spacing: time.Hour, Platforms: both, Budget: budget}

func TestWithNoReviewARunsClipsGoOutAtOnceAndASpacingApartAfterWhatThePlatformsHold(t *testing.T) {
	posted := &feeds{feed: posts.Feed{{At: now.Add(-30 * time.Minute), Platforms: both}}}
	r := schedulingWith(asap, posted, folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []time.Time{now.Add(30 * time.Minute), now.Add(90 * time.Minute)}, []time.Time{report[0].At, report[1].At})
	assert.Equal(t, []time.Time{now.Add(-24 * time.Hour)}, posted.since)
	assert.True(t, report.Complete())
}

func TestAClipNoPlatformHasRoomForIsHeldAndTheRunIsStillComplete(t *testing.T) {
	full := make(posts.Feed, 0, 9)
	for i := range 9 {
		full = append(full, posts.Posting{At: now.Add(-time.Duration(i+2) * time.Hour), Platforms: []posts.Platform{posts.TikTok}})
	}
	r := schedulingWith(asap, &feeds{feed: full}, folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	require.Len(t, r.service.Scheduled, 2)
	assert.Equal(t, both, r.service.Scheduled[0].Platforms)
	// TikTok's tenth went out with BEST: SECOND goes to YouTube alone.
	assert.Equal(t, []posts.Platform{posts.YouTube}, r.service.Scheduled[1].Platforms)
	assert.False(t, report[1].Held)

	tight := asap
	tight.Budget = posts.Budget{posts.TikTok: 10}
	r = schedulingWith(tight, &feeds{feed: full}, folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}})

	report, err = r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Len(t, r.service.Scheduled, 1)
	assert.Equal(t, posts.Outcome{Title: "SECOND", Held: true}, report[1])
	assert.True(t, report.Complete())
}

func TestAPostAWaitMovedLaterMovesTheNextOneToo(t *testing.T) {
	clock := cptime.NewFixedClock(now)
	service := posts.NewScriptedService(&posts.BusyError{RetryAfter: 10 * time.Minute, Reason: "rate limit"})
	sender := posts.NewSender(service, posts.Backoff{15 * time.Minute}, posts.NewRecordedPause(clock), clock)
	usecase := schedule_clips_usecase.New(folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}}, &feeds{}, asap,
		clock, posts.LoadedDice(1), sender)

	report, err := usecase.Execute(t.Context())

	require.NoError(t, err)
	// Planned at 12:45, sent after a 15-minute wait: 13:00, and SECOND an hour after it.
	assert.Equal(t, []time.Time{now.Add(20 * time.Minute), now.Add(80 * time.Minute)}, []time.Time{report[0].At, report[1].At})
}

func TestAFeedThatCannotBeReadFailsTheRunBeforeAnythingIsSent(t *testing.T) {
	unreachable := errors.New("zernio could not be reached")
	r := schedulingWith(asap, &feeds{err: unreachable}, folder{clips: []posts.Clip{clip(t, 1, "BEST")}})

	_, err := r.usecase.Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
	assert.Empty(t, r.service.Scheduled)
}
