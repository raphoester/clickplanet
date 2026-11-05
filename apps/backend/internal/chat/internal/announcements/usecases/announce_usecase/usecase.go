package announce_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
)

type Appender interface {
	AppendOnce(ctx context.Context, announcement announcements.Announcement) (bool, error)
}

type Publisher interface {
	Publish(update feed.Update)
}

func New(appender Appender, publisher Publisher) *UseCase {
	return &UseCase{appender: appender, publisher: publisher}
}

type UseCase struct {
	appender  Appender
	publisher Publisher
}

func (u *UseCase) Execute(ctx context.Context, announcement announcements.Announcement) error {
	if !announcement.Kind().Known() {
		return fmt.Errorf("%w: %q", announcements.ErrUnknownKind, announcement.Kind())
	}

	appended, err := u.appender.AppendOnce(ctx, announcement)
	if err != nil {
		return fmt.Errorf("failed to store a %s announcement: %w", announcement.Kind(), err)
	}
	if appended {
		u.publisher.Publish(feed.Announced(announcement))
	}
	return nil
}
