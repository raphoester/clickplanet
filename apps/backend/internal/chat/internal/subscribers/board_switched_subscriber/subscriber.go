package board_switched_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, announcement announcements.Announcement) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.BoardSwitched] = Subscriber{}

var errNoTime = errors.New("the switch has no time")

// Handle announces the world's switch of board in the world chat, once: an arena plays the board its set names and is
// no switch.
func (s Subscriber) Handle(ctx context.Context, event *planetv1.BoardSwitched) error {
	if err := event.GetSwitchedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}
	if event.GetMap() != "world" {
		return nil
	}
	announcement, err := announcements.SwitchOf(event.GetPosition(), event.GetFrom(), event.GetTo(), event.GetFromTiles(),
		event.GetToTiles()).Announcement(event.GetSwitchedAt().AsTime())
	if err != nil {
		return err //nolint:wrapcheck // Announcement named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()
	return s.useCase.Execute(ctx, announcement) //nolint:wrapcheck // the use case named it.
}
