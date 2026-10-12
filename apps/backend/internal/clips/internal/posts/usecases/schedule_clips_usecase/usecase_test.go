package schedule_clips_usecase_test

import (
	"context"
	"errors"
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
	cadence = posts.Cadence{Review: 4 * time.Hour, Spacing: 2 * time.Hour}
	first   = now.Add(4 * time.Hour)
	second  = now.Add(6 * time.Hour)
)

type folder struct {
	clips []posts.Clip
	err   error
}

func (f folder) Rendered(context.Context) ([]posts.Clip, error) {
	return f.clips, f.err
}

type page struct {
	written []posts.Report
	err     error
}

func (p *page) Write(report posts.Report) error {
	p.written = append(p.written, report)
	return p.err
}

func clip(t *testing.T, pick posts.Pick, title string) posts.Clip {
	t.Helper()

	restored, err := posts.ClipSnapshot{Pick: pick, Title: title, Caption: title + " 👇", Video: "clip.mp4"}.Restore()
	require.NoError(t, err)
	return restored
}

type run struct {
	service *posts.ScriptedService
	summary *page
	usecase *schedule_clips_usecase.UseCase
}

func scheduling(clips folder, refusals ...error) run {
	clock := cptime.NewFixedClock(now)
	service := posts.NewScriptedService(refusals...)
	summary := &page{}
	sender := posts.NewSender(service, posts.Backoff{time.Minute}, posts.NewRecordedPause(clock), clock)
	return run{
		service: service, summary: summary,
		usecase: schedule_clips_usecase.New(clips, cadence, clock, sender, summary),
	}
}

func TestEachRenderedClipIsScheduledBestFirstAndTheSummarySaysSo(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 2, "SECOND"), clip(t, 1, "BEST")}})

	report, err := r.usecase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first, Post: "post-BEST"},
		{Title: "SECOND", At: second, Post: "post-SECOND"},
	}, report)
	assert.Equal(t, []posts.Report{report}, r.summary.written)
	assert.Equal(t, []posts.PostSnapshot{
		{Title: "BEST", Caption: "BEST 👇", Video: "clip.mp4", At: first},
		{Title: "SECOND", Caption: "SECOND 👇", Video: "clip.mp4", At: second},
	}, r.service.Scheduled)
}

func TestAClipTheServiceRefusesFailsTheRunAndTheOthersStillGoOut(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST"), clip(t, 2, "SECOND")}}, errors.New("the caption is too long"))

	report, err := r.usecase.Execute(t.Context())

	require.ErrorIs(t, err, posts.ErrUnsent)
	assert.Equal(t, posts.Report{
		{Title: "BEST", At: first, Failure: "the caption is too long"},
		{Title: "SECOND", At: second, Post: "post-SECOND"},
	}, report)
	assert.Equal(t, []posts.Report{report}, r.summary.written)
}

func TestARunWithNoRenderedClipFailsAndWritesNoSummary(t *testing.T) {
	r := scheduling(folder{})

	_, err := r.usecase.Execute(t.Context())

	require.ErrorIs(t, err, schedule_clips_usecase.ErrNoClip)
	assert.Empty(t, r.summary.written)
}

func TestClipsThatCannotBeReadFailTheRun(t *testing.T) {
	unreadable := errors.New("the folder is gone")
	r := scheduling(folder{err: unreadable})

	_, err := r.usecase.Execute(t.Context())

	require.ErrorIs(t, err, unreadable)
	assert.Empty(t, r.service.Scheduled)
}

func TestASummaryThatCannotBeWrittenFailsTheRunAfterTheClipsWentOut(t *testing.T) {
	r := scheduling(folder{clips: []posts.Clip{clip(t, 1, "BEST")}})
	r.summary.err = errors.New("the disk is full")

	report, err := r.usecase.Execute(t.Context())

	require.ErrorIs(t, err, r.summary.err)
	assert.Len(t, report, 1)
	assert.Len(t, r.service.Scheduled, 1)
}
