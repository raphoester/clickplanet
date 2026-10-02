package announce_usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
)

type Appender interface {
	Append(ctx context.Context, announcement announcements.Announcement) error
}

type Publisher interface {
	Publish(update feed.Update)
}

type In struct {
	Kind    announcements.Kind
	At      time.Time
	Payload json.RawMessage
}

func New(appender Appender, publisher Publisher) *UseCase {
	return &UseCase{appender: appender, publisher: publisher}
}

type UseCase struct {
	appender  Appender
	publisher Publisher
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	announcement := announcements.Announcement{
		ID:      announcements.AnnouncementID(uuid.New()),
		Kind:    in.Kind,
		At:      in.At,
		Payload: in.Payload,
	}

	if err := u.appender.Append(ctx, announcement); err != nil {
		return fmt.Errorf("failed to store a %s announcement: %w", in.Kind, err)
	}

	u.publisher.Publish(feed.Update{Announcement: &announcement})
	return nil
}
