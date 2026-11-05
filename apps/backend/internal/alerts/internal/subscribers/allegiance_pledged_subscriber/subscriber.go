package allegiance_pledged_subscriber

import (
	"context"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/follow_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in follow_flag_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.AllegiancePledged] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *planetv1.AllegiancePledged) error {
	account, err := links.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the rule named it.
	}
	flag, err := nations.FlagOf(event.GetCountry())
	if err != nil {
		return err //nolint:wrapcheck // the flag named it.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, follow_flag_usecase.In{Account: account, Flag: flag}) //nolint:wrapcheck // the use case named it.
}
