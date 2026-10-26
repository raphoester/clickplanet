package take_snapshot_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Source interface {
	// The connected accounts, and whether the service gives analytics at all.
	Accounts(ctx context.Context) (stats.Accounts, error)
	// Every post of the clips job scheduled at since or later, and maybe some before.
	Posts(ctx context.Context, since time.Time) ([]stats.Post, error)
	// stats.ErrNoAnalyticsAccess when the service gives none.
	Analytics(ctx context.Context, post stats.PostID) (stats.Answer, error)
	// The curve of a YouTube video, as of the day given: any answer YouTube gave is a Retention, and an error is none.
	Retention(ctx context.Context, account, video string, asOf time.Time) (stats.Retention, error)
}

type Store interface {
	// The report says what the rows are of: when, and what was read.
	Keep(ctx context.Context, report stats.Report, rows []stats.Row) error
	// The rows of the last snapshot kept, none before the first: the curves they hold are carried until the next is due.
	Last(ctx context.Context) ([]stats.Row, error)
}

type Executor interface {
	Execute(ctx context.Context) (stats.Report, error)
}

// since: zero is stats.DefaultSince.
func New(source Source, store Store, clock cptime.Clock, since time.Time, backoff posts.Backoff, pause posts.Pause) *UseCase {
	return &UseCase{source: source, store: store, clock: clock, since: since, backoff: backoff, pause: pause}
}

type UseCase struct {
	source  Source
	store   Store
	clock   cptime.Clock
	since   time.Time
	backoff posts.Backoff
	pause   posts.Pause
}

var _ Executor = (*UseCase)(nil)

// What could be read is kept even when a post could not be: a day not kept is lost for good. The report says which.
func (u *UseCase) Execute(ctx context.Context) (stats.Report, error) {
	takenAt := u.clock.Now().UTC().Truncate(time.Second)
	since := u.since
	if since.IsZero() {
		since = stats.DefaultSince(takenAt)
	}

	accounts, err := u.source.Accounts(ctx)
	if err != nil {
		return stats.Report{}, fmt.Errorf("failed to read the connected accounts: %w", err)
	}
	if !accounts.AnalyticsAccess {
		return stats.Report{}, stats.ErrNoAnalyticsAccess
	}

	all, err := u.source.Posts(ctx, since)
	if err != nil {
		return stats.Report{}, fmt.Errorf("failed to read the posts of the job: %w", err)
	}

	last, err := u.store.Last(ctx)
	if err != nil {
		return stats.Report{}, fmt.Errorf("failed to read the last snapshot: %w", err)
	}
	curves := &curves{UseCase: u, held: stats.HeldRetentions(last), takenAt: takenAt}

	var rows []stats.Row
	var unread []stats.Unread
	notOut := 0
	for _, post := range all {
		if !post.Out() {
			notOut++
			continue
		}
		postRows, failure, err := u.snapshot(ctx, post, curves, takenAt)
		if err != nil {
			return stats.Report{}, err
		}
		rows = append(rows, postRows...)
		if failure != "" {
			unread = append(unread, stats.Unread{PostID: post.ID, Reason: failure})
		}
	}

	report := stats.ReportOf(accounts, rows)
	report.TakenAt, report.Since, report.Posts, report.NotOut, report.Unread = takenAt, since, len(all), notOut, unread
	report.RetentionAsked = curves.asked

	if err := u.store.Keep(ctx, report, rows); err != nil {
		return stats.Report{}, fmt.Errorf("failed to keep the snapshot: %w", err)
	}
	return report, nil
}

// The rows of one post, and what of it could not be read. The error is for the run: the add-on is off, or it was
// cancelled. A curve that could not be read loses nothing else: the row says why it is missing.
func (u *UseCase) snapshot(ctx context.Context, post stats.Post, curves *curves, takenAt time.Time) ([]stats.Row, string, error) {
	var answer stats.Answer
	err := u.retried(ctx, func() (err error) {
		answer, err = u.source.Analytics(ctx, post.ID)
		return err //nolint:wrapcheck // a pure delegation: the service already named what failed.
	})
	if err != nil {
		if errors.Is(err, stats.ErrNoAnalyticsAccess) || ctx.Err() != nil {
			return nil, "", err
		}
		return nil, err.Error(), nil
	}

	retentions := map[string]stats.RetentionState{}
	for _, target := range post.Targets {
		if target.Platform != string(posts.YouTube) || !target.Out() || target.PlatformPostID == "" {
			continue
		}
		state, err := curves.of(ctx, target)
		if err != nil {
			return nil, "", err
		}
		retentions[target.PlatformPostID] = state
	}
	return stats.RowsOf(post, answer, retentions, takenAt), "", nil
}

// The curves of one run: those the last snapshot held, and why none is asked any more, once YouTube said it would give
// none for a while (its daily quota spent, until midnight Pacific): each would be a call refused at once.
type curves struct {
	*UseCase
	held     map[string]*stats.Retention
	takenAt  time.Time
	asked    int
	withheld string
}

// The curve a video's row holds: read when one is due, else carried. The error is the context's alone.
func (c *curves) of(ctx context.Context, target stats.Target) (stats.RetentionState, error) {
	held := c.held[target.PlatformPostID]
	if !stats.RetentionDue(c.takenAt.Sub(target.PublishedAt), held) {
		return stats.RetentionState{Retention: held}, nil
	}
	if c.withheld != "" {
		return stats.RetentionState{Retention: held, Missing: c.withheld}, nil
	}

	var retention stats.Retention
	err := c.retried(ctx, func() (err error) {
		c.asked++
		retention, err = c.source.Retention(ctx, target.AccountID, target.PlatformPostID, c.takenAt)
		return err //nolint:wrapcheck // a pure delegation: the service already named what failed.
	})
	switch {
	case err == nil:
		return stats.RetentionAfter(held, retention.ReadOn(target, c.takenAt)), nil
	case ctx.Err() != nil:
		return stats.RetentionState{}, ctx.Err() //nolint:wrapcheck // the context's own error says why the run ended.
	case errors.Is(err, stats.ErrRetentionQuota):
		c.withheld = stats.QuotaSpent
		return stats.RetentionState{Retention: held, Missing: stats.QuotaSpent}, nil
	case errors.As(err, new(*posts.BusyError)):
		// Busy for longer than the backoff waits: the quota, in words Zernio did not use on 2026-10-25.
		c.withheld = "not asked: " + err.Error()
		return stats.RetentionState{Retention: held, Missing: err.Error()}, nil
	default:
		return stats.RetentionState{Retention: held, Missing: err.Error()}, nil
	}
}

// Tries again while the service is busy, as the sender does.
func (u *UseCase) retried(ctx context.Context, try func() error) error {
	for failed := 1; ; failed++ {
		err := try()
		var busy *posts.BusyError
		if !errors.As(err, &busy) {
			return err
		}
		wait, again := u.backoff.Wait(failed, busy.RetryAfter)
		if !again {
			return err
		}
		if err := u.pause.Wait(ctx, wait); err != nil {
			return err //nolint:wrapcheck // a pure delegation: the pause only ends early with the context.
		}
	}
}
