package tell_gone_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
)

type Streams interface {
	Publish(flag feed.Told)
}

type In struct {
	Flag      string
	DeletedAt time.Time
}

func New(news feed.News, streams Streams) *UseCase {
	return &UseCase{news: news, streams: streams}
}

type UseCase struct {
	news    feed.News
	streams Streams
}

func (u *UseCase) Execute(_ context.Context, in In) error {
	if !u.news.Fresh(in.DeletedAt) {
		return nil
	}
	id, err := catalog.ParseFlagID(in.Flag)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	u.streams.Publish(feed.ToldGone(id))
	return nil
}
