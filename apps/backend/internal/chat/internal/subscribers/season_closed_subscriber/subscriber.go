package season_closed_subscriber

import (
	"context"
	"errors"
	"fmt"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_champion_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type UseCase interface {
	Execute(ctx context.Context, in announce_champion_usecase.In) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpbootstrap.Handler[*seasonsv1.SeasonClosed] = Subscriber{}

var errNoTime = errors.New("the season has no end")

func (s Subscriber) Handle(ctx context.Context, event *seasonsv1.SeasonClosed) error {
	if err := event.GetEndedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %w", errNoTime, err)
	}

	table := make([]announcements.Place, 0, len(event.GetTable()))
	for _, place := range event.GetTable() {
		table = append(table, announcements.PlaceOf(place.GetCountry(), place.GetRank(), place.GetPoints()))
	}
	champion := announcements.ChampionOf(event.GetSeason(), table)
	if !champion.Crowned() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, announce_champion_usecase.In{ //nolint:wrapcheck // the use case named it.
		Champion: champion,
		At:       event.GetEndedAt().AsTime(),
	})
}
