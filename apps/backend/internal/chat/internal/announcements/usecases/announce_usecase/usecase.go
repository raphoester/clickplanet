package announce_usecase

import (
	"context"
	"encoding/json"
	"errors"
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

	// Set, the line is kept and shown once however often its event comes: its id is drawn from the key.
	Once string
}

var onceSpace = uuid.MustParse("8b0e7c52-5f2d-4c1e-9a64-3f1d2b7a9e10")

func New(appender Appender, publisher Publisher) *UseCase {
	return &UseCase{appender: appender, publisher: publisher}
}

type UseCase struct {
	appender  Appender
	publisher Publisher
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	if !in.Kind.Known() {
		return fmt.Errorf("%w: %q", announcements.ErrUnknownKind, in.Kind)
	}

	id := uuid.New()
	if in.Once != "" {
		id = uuid.NewSHA1(onceSpace, []byte(string(in.Kind)+":"+in.Once))
	}
	announcement := announcements.NewAnnouncement(announcements.AnnouncementID(id), in.Kind, in.At, in.Payload)

	if err := u.appender.Append(ctx, announcement); err != nil {
		if errors.Is(err, announcements.ErrKept) && in.Once != "" {
			return nil
		}
		return fmt.Errorf("failed to store a %s announcement: %w", in.Kind, err)
	}

	u.publisher.Publish(feed.Announced(announcement))
	return nil
}
