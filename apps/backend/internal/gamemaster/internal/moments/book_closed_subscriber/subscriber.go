package book_closed_subscriber

import (
	"context"
	"time"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const timeout = 5 * time.Second

func New(useCase hear_book_usecase.UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase hear_book_usecase.UseCase
}

var _ cpredis.Handler[*scoringv1.BookClosed] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *scoringv1.BookClosed) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := s.useCase.Execute(ctx, moments.ClosedBook{
		Book: moments.BookID(event.GetBookId()), EndedAt: event.GetEndedAt().AsTime(),
	})
	return err //nolint:wrapcheck // the use case named it.
}
