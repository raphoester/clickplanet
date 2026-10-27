package play_season_handler

import (
	"context"
	"errors"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, decision calendar.Decision) (bool, error)
}

func New(useCase UseCase) Handler {
	return Handler{useCase: useCase}
}

type Handler struct {
	useCase UseCase
}

var _ cpredis.Handler[*scoringv1.PlaySeason] = Handler{}

func (h Handler) Handle(ctx context.Context, command *scoringv1.PlaySeason) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	_, err := h.useCase.Execute(ctx, calendar.Decision{
		Window: calendar.Window{
			Number: calendar.Number(command.GetNumber()), StartsAt: command.GetStartsAt().AsTime(), EndsAt: command.GetEndsAt().AsTime(),
		},
		DecidedAt: command.GetDecidedAt().AsTime(),
	})
	// A window refused is logged and dropped: delivered again, it would hold back the next decision.
	if errors.Is(err, calendar.ErrInvalidSeason) {
		return nil
	}
	return err //nolint:wrapcheck // the use case named it.
}
