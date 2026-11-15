package box_claimed_subscriber

import (
	"context"
	"errors"
	"fmt"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_caught_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, box bonuses.CaughtBox) (pay_caught_box_usecase.Out, error)
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

// Subscriber pays each box the boxes tell caught.
type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*boxesv1.BoxClaimed] = Subscriber{}

var errNoToken = errors.New("the catch names no box")

func (s Subscriber) Handle(ctx context.Context, event *boxesv1.BoxClaimed) error {
	account, err := bonuses.ParseAccount(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("box claimed: %w", err)
	}
	if event.GetToken() == "" {
		return errNoToken
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	_, err = s.useCase.Execute(ctx, bonuses.CaughtBox{
		Account: account, Token: event.GetToken(), Welcome: event.GetWelcome(),
	})
	return err //nolint:wrapcheck // the use case named it.
}
