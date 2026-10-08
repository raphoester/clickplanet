package lead_changed_subscriber

import (
	"context"
	"errors"
	"fmt"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, in announce_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*seasonsv1.LeadChanged] = Subscriber{}

var (
	errNoTime    = errors.New("the lead change has no time")
	errNoCountry = errors.New("the lead change names no country")
)

func (s Subscriber) Handle(ctx context.Context, event *seasonsv1.LeadChanged) error {
	if err := event.GetChangedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}
	if event.GetLeader() == "" || event.GetPassed() == "" {
		return errNoCountry
	}

	payload, err := announcements.LeadChangeOf(event.GetSeason(), event.GetLeader(), event.GetPassed()).Payload()
	if err != nil {
		return err //nolint:wrapcheck // Payload named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_usecase.In{ //nolint:wrapcheck // the use case named it.
		Kind:    announcements.KindLeadChanged,
		At:      event.GetChangedAt().AsTime(),
		Payload: payload,
	})
}
