package season_played_subscriber

import (
	"context"
	"errors"
	"fmt"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/usecases/record_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, in record_season_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*seasonsv1.SeasonPlayed] = Subscriber{}

var errNoFlag = errors.New("the season has no flag")

func (s Subscriber) Handle(ctx context.Context, event *seasonsv1.SeasonPlayed) error {
	account, err := players.AccountIDOf(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("season %d: %w", event.GetSeason(), err)
	}
	if event.GetMainFlag() == "" {
		return fmt.Errorf("season %d: %w", event.GetSeason(), errNoFlag)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_season_usecase.In{Account: account, Season: seasons.Season{ //nolint:wrapcheck // the use case named it.
		Number: seasons.Number(event.GetSeason()),
		Flag:   seasons.Country(event.GetMainFlag()),
		Won:    event.GetWon(),
	}})
}
