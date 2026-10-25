package announce_champion_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
)

type Appender interface {
	AppendOnce(ctx context.Context, announcement announcements.Announcement) (bool, error)
}

type Publisher interface {
	Publish(update feed.Update)
}

type In struct {
	Champion announcements.Champion
	At       time.Time
}

func New(appender Appender, publisher Publisher) *UseCase {
	return &UseCase{appender: appender, publisher: publisher}
}

type UseCase struct {
	appender  Appender
	publisher Publisher
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	announcement, err := in.Champion.Announcement(in.At)
	if err != nil {
		return err //nolint:wrapcheck // Announcement named it.
	}

	appended, err := u.appender.AppendOnce(ctx, announcement)
	if err != nil {
		return fmt.Errorf("failed to store a champion announcement: %w", err)
	}
	if appended {
		u.publisher.Publish(feed.Announced(announcement))
	}
	return nil
}
