package announce_usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func New(appender Appender, publisher Publisher, ids announcements.IDProvider) *UseCase {
	return &UseCase{appender: appender, publisher: publisher, ids: ids}
}

type UseCase struct {
	appender  Appender
	publisher Publisher
	ids       announcements.IDProvider
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	if !in.Kind.Known() {
		return fmt.Errorf("%w: %q", announcements.ErrUnknownKind, in.Kind)
	}

	id, err := u.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to draw an announcement id: %w", err)
	}

	announcement := announcements.NewAnnouncement(id, in.Kind, in.At, in.Payload)

	if err := u.appender.Append(ctx, announcement); err != nil {
		return fmt.Errorf("failed to store a %s announcement: %w", in.Kind, err)
	}

	u.publisher.Publish(feed.Announced(announcement))
	return nil
}
