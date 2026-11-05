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

	var rows []stats.Row
	var unread []stats.Unread
	notOut := 0
	// Why YouTube's curves stopped being asked for, once it said it would give none for a while.
	var curvesWithheld error
	for _, post := range all {
		if !post.Out() {
			notOut++
			continue
		}
		postRows, failure, err := u.snapshot(ctx, post, takenAt, &curvesWithheld)
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

	if err := u.store.Keep(ctx, report, rows); err != nil {
		return stats.Report{}, fmt.Errorf("failed to keep the snapshot: %w", err)
	}
	return report, nil
}

// The rows of one post, and what of it could not be read. The error is for the run: the add-on is off, or it was
// cancelled. withheld: once a curve is refused as busy for longer than the backoff waits (YouTube's daily quota spent,
// until midnight Pacific), no other is asked for in this run, each a call refused at once that may count against it.
func (u *UseCase) snapshot(ctx context.Context, post stats.Post, takenAt time.Time, withheld *error) ([]stats.Row, string, error) {
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

	var failures []error
	retentions := map[string]*stats.Retention{}
	for _, target := range post.Targets {
		if target.Platform != string(posts.YouTube) || !target.Out() || target.PlatformPostID == "" {
			continue
		}
		if *withheld != nil {
			failures = append(failures, fmt.Errorf("the retention of %s: not asked: %w", target.PlatformPostID, *withheld))
			continue
		}
		var retention stats.Retention
		err := u.retried(ctx, func() (err error) {
			retention, err = u.source.Retention(ctx, target.AccountID, target.PlatformPostID, takenAt)
			return err //nolint:wrapcheck // a pure delegation: the service already named what failed.
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, "", ctx.Err() //nolint:wrapcheck // the context's own error says why the run ended.
			}
			failures = append(failures, fmt.Errorf("the retention of %s: %w", target.PlatformPostID, err))
			if busy := (*posts.BusyError)(nil); errors.As(err, &busy) {
				*withheld = err
			}
			continue
		}
		retentions[target.PlatformPostID] = &retention
	}

	rows := stats.RowsOf(post, answer, retentions, takenAt)
	if len(failures) > 0 {
		return rows, errors.Join(failures...).Error(), nil
	}
	return rows, "", nil
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
