package listen_for_titles_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

type Feed interface {
	Subscribe(ctx context.Context, account players.AccountID) <-chan titles.IDs
}

type Catalog interface {
	Shown(held titles.IDs) []titles.Standing
}

type Sink interface {
	SendTitleEarned(standing titles.Standing) error
}

type UseCase struct {
	feed    Feed
	catalog Catalog
}

func New(feed Feed, catalog Catalog) *UseCase {
	return &UseCase{feed: feed, catalog: catalog}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID, sink Sink) error {
	for earned := range u.feed.Subscribe(ctx, account) {
		for _, standing := range u.catalog.Shown(earned) {
			if err := sink.SendTitleEarned(standing); err != nil {
				return err //nolint:wrapcheck // the stream's own error.
			}
		}
	}
	return nil
}
