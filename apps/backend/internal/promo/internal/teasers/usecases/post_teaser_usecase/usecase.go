package post_teaser_usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The battles to come with their cards: listed_battles.
type Battles interface {
	Teasers(ctx context.Context, within time.Duration) ([]teasers.Teaser, error)
}

// The publishing module's posts: rpc_publishing.
type Posts interface {
	// The battles the posts since then tease.
	Teased(ctx context.Context, since time.Time) ([]teasers.BattleID, error)
}

// teasers.Poster.
type Poster interface {
	Post(ctx context.Context, post teasers.Post) (teasers.Outcome, error)
}

type Executor interface {
	// What became of the teaser is the outcome. teasers.ErrInvalidPost for media that are not the battle's.
	Execute(ctx context.Context, battle teasers.BattleID, media teasers.Media) (teasers.Outcome, error)
}

func New(battles Battles, posts Posts, poster Poster, clock cptime.Clock) *UseCase {
	return &UseCase{battles: battles, posts: posts, poster: poster, clock: clock}
}

type UseCase struct {
	battles Battles
	posts   Posts
	poster  Poster
	clock   cptime.Clock
}

var _ Executor = (*UseCase)(nil)

// The battle's teaser posted, if it is still due: the battles and the posts are read again, as either may have changed
// while the job rendered its cards.
func (u *UseCase) Execute(ctx context.Context, battle teasers.BattleID, media teasers.Media) (teasers.Outcome, error) {
	listed, err := u.battles.Teasers(ctx, teasers.Ahead)
	if err != nil {
		return teasers.Outcome{}, fmt.Errorf("failed to list the battles to come: %w", err)
	}
	index := slices.IndexFunc(listed, func(teaser teasers.Teaser) bool { return teaser.Battle == battle })
	if index < 0 {
		return teasers.NotPosted(teasers.Skipped, "no longer on the schedule within 36 hours"), nil
	}

	teased, err := u.posts.Teased(ctx, teasers.TeasedSince(u.clock.Now()))
	var busy *teasers.BusyError
	switch {
	case errors.As(err, &busy):
		return teasers.BusyOf(busy), nil
	case err != nil:
		return teasers.Outcome{}, fmt.Errorf("failed to read the battles teased already: %w", err)
	case slices.Contains(teased, battle):
		return teasers.NotPosted(teasers.Skipped, "teased already"), nil
	}

	post, err := teasers.NewPost(listed[index], media)
	if err != nil {
		return teasers.Outcome{}, err //nolint:wrapcheck // the domain already says what is wrong with the media.
	}
	return u.poster.Post(ctx, post) //nolint:wrapcheck // the poster already named what failed.
}
