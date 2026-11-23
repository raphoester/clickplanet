package tell_streams_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
)

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
}

type Streams interface {
	Publish(flag catalog.Snapshot)
}

type In struct {
	Flag      string
	DecidedAt time.Time
}

const timeout = 5 * time.Second

func New(news feed.News, flags Flags, streams Streams) *UseCase {
	return &UseCase{news: news, flags: flags, streams: streams}
}

type UseCase struct {
	news    feed.News
	flags   Flags
	streams Streams
}

// Execute tells the open streams the flag as the catalog holds it now, whichever of its facts came last.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	if !u.news.Fresh(in.DecidedAt) {
		return nil
	}
	id, err := catalog.ParseFlagID(in.Flag)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	flag, err := u.flags.Flag(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to read the flag to tell: %w", err)
	}
	u.streams.Publish(flag.TakeSnapshot())
	return nil
}
