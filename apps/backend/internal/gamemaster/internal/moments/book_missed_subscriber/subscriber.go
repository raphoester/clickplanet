package book_missed_subscriber

import (
	"context"
	"time"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_missed_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const timeout = 5 * time.Second

func New(useCase hear_missed_book_usecase.UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase hear_missed_book_usecase.UseCase
}

var _ cpredis.Handler[*scoringv1.BookMissed] = Subscriber{}

// An event that names no book is dropped: delivered again, it would hold back the ones after it.
func (s Subscriber) Handle(ctx context.Context, event *scoringv1.BookMissed) error {
	if event.GetBookId() == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := s.useCase.Execute(ctx, moments.MissedBook{
		Book: moments.BookID(event.GetBookId()), EndedAt: event.GetEndedAt().AsTime(),
	})
	return err //nolint:wrapcheck // the use case named it.
}
