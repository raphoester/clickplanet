package flag_knocked_out_subscriber

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

var _ cpredis.Handler[*planetv1.FlagKnockedOut] = Subscriber{}

var errNoTime = errors.New("the knock-out has no time")

func (s Subscriber) Handle(ctx context.Context, event *planetv1.FlagKnockedOut) error {
	if err := event.GetKnockedOutAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	announcement, err := announcements.KnockOutOf(event.GetPosition(), event.GetCountry(), event.GetBy()).
		Announcement(event.GetKnockedOutAt().AsTime())
	if err != nil {
		return err //nolint:wrapcheck // Announcement named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announcement) //nolint:wrapcheck // the use case named it.
}
