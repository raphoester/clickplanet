package season_podium_subscriber

import (
	"context"
	"time"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums/usecases/record_podium_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in record_podium_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*gamemasterv1.SeasonEnded] = Subscriber{}

const timeout = time.Minute

func (s Subscriber) Handle(ctx context.Context, event *gamemasterv1.SeasonEnded) error {
	book, err := finales.BookIDOf(event.GetBookId())
	if err != nil {
		return err //nolint:wrapcheck // BookIDOf named it.
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return s.useCase.Execute(ctx, record_podium_usecase.In{ //nolint:wrapcheck // the use case named it.
		Season: finales.Season(event.GetSeason()),
		Book:   book,
	})
}
