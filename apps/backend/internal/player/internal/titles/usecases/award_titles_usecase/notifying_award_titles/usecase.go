package notifying_award_titles

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/award_titles_usecase"
)

type Feed interface {
	Publish(account players.AccountID, earned titles.IDs)
}

func New(inner award_titles_usecase.Executor, feed Feed) *Decorator {
	return &Decorator{inner: inner, feed: feed}
}

type Decorator struct {
	inner award_titles_usecase.Executor
	feed  Feed
}

var _ award_titles_usecase.Executor = (*Decorator)(nil)

func (d *Decorator) Execute(ctx context.Context, account players.AccountID) (titles.IDs, error) {
	earned, err := d.inner.Execute(ctx, account)
	if err != nil || len(earned) == 0 {
		return earned, err //nolint:wrapcheck // a decorator adds a notification, not a sentence.
	}

	d.feed.Publish(account, earned)
	return earned, nil
}
