package schedule_clips_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoClip = errors.New("no clip was rendered")

type Clips interface {
	Rendered(ctx context.Context) ([]posts.Clip, error)
}

type Sender interface {
	// It answers the post as it was sent, which may be later than planned.
	Send(ctx context.Context, post posts.Post) (posts.Post, posts.Scheduled, error)
}

type Feeds interface {
	// Every post, by hand or not, that goes out at since or later on each platform, and maybe some before.
	Posted(ctx context.Context, since time.Time) (posts.Feed, error)
}

type Charts interface {
	// TikTok's trending tracks in the country, rank 1 first.
	Trending(ctx context.Context, country string) ([]posts.Track, error)
}

type Executor interface {
	// A clip the service refused is in the report, not in the error.
	Execute(ctx context.Context) (posts.Report, error)
}

// experiments: the ones that run, which every clip's arms are checked against before any is posted. charts and
// soundtracks: where a clip drawn for a trending track gets one, and how.
func New(
	clips Clips, feeds Feeds, plan posts.Plan, clock cptime.Clock, dice posts.Dice, sender Sender, experiments posts.Experiments,
	charts Charts, soundtracks posts.Soundtracks,
) *UseCase {
	return &UseCase{
		clips: clips, feeds: feeds, plan: plan, clock: clock, dice: dice, sender: sender, experiments: experiments,
		charts: charts, soundtracks: soundtracks,
	}
}

type UseCase struct {
	clips       Clips
	feeds       Feeds
	plan        posts.Plan
	clock       cptime.Clock
	dice        posts.Dice
	sender      Sender
	experiments posts.Experiments
	charts      Charts
	soundtracks posts.Soundtracks
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (posts.Report, error) {
	rendered, err := u.clips.Rendered(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the rendered clips: %w", err)
	}
	if len(rendered) == 0 {
		return nil, ErrNoClip
	}
	// Before any post: a clip in an arm nobody runs fails the run, rather than be counted wrong once it is out.
	for _, clip := range rendered {
		snapshot := clip.TakeSnapshot()
		if err := u.experiments.Check(snapshot.Variants); err != nil {
			return nil, fmt.Errorf("the clip %q: %w", snapshot.Title, err)
		}
	}

	feed, err := u.feeds.Posted(ctx, posts.PostedSince(u.clock.Now()))
	if err != nil {
		return nil, fmt.Errorf("failed to read what each platform holds: %w", err)
	}

	// Each post is planned once the one before it is sent: a wait that moved it moves the next one too.
	report := posts.Report{}
	ranked := posts.Ranked(rendered)
	chart := u.chartOnce()
	for i, clip := range ranked {
		post, room := u.plan.Next(clip, i+1, len(ranked), feed, u.clock.Now(), u.dice)
		if !room {
			report = report.WithHeld(clip)
			continue
		}
		if post.WantsTrendingSound() {
			post = u.withTrendingSound(ctx, post, clip, chart)
		}
		sent, scheduled, err := u.sender.Send(ctx, post)
		report = report.With(sent, scheduled, err)
		feed = feed.With(sent, scheduled)
	}
	return report, nil
}

// The chart is read once a run, when a post first needs it, and each post picks its own track from it.
func (u *UseCase) chartOnce() func(ctx context.Context) ([]posts.Track, error) {
	var (
		read  bool
		chart []posts.Track
		err   error
	)
	return func(ctx context.Context) ([]posts.Track, error) {
		if !read {
			read = true
			if chart, err = u.charts.Trending(ctx, u.soundtracks.Country); err != nil {
				err = fmt.Errorf("the trending chart could not be read: %w", err)
			}
		}
		return chart, err
	}
}

// A chart that cannot be read, or lists nothing to attach that plays the whole clip, never fails the run: the clip
// goes out as the control would, counted apart (posts.FellBack), and the report says why.
func (u *UseCase) withTrendingSound(
	ctx context.Context, post posts.Post, clip posts.Clip, chart func(ctx context.Context) ([]posts.Track, error),
) posts.Post {
	tracks, err := chart(ctx)
	if err != nil {
		return post.FellBack(posts.SoundExperiment, err)
	}
	sound, err := u.soundtracks.Pick(tracks, clip, u.dice)
	if err != nil {
		return post.FellBack(posts.SoundExperiment, err)
	}
	return post.WithSound(sound)
}
