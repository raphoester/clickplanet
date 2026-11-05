package grant_points_handler

import (
	"context"
	"fmt"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/grant_points_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Executor interface {
	Execute(ctx context.Context, command grant_points_usecase.Command) (bool, error)
}

func New(useCase Executor) Handler {
	return Handler{useCase: useCase}
}

// Handler parses each grant for its use case, and answers every error, a refusal too.
type Handler struct {
	useCase Executor
}

var _ cpredis.Handler[*scoringv1.GrantPoints] = Handler{}

func (h Handler) Handle(ctx context.Context, command *scoringv1.GrantPoints) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	grant, err := books.ParseGrantID(command.GetGrantId())
	if err != nil {
		return fmt.Errorf("%w: %w", books.ErrRefused, err)
	}
	if err := command.GetGrantedAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: grant %q has no time: %w", books.ErrRefused, grant, err)
	}
	_, err = h.useCase.Execute(ctx, grant_points_usecase.Command{
		Grant: grant, Country: books.Country(command.GetCountry()), Points: command.GetPoints(), At: command.GetGrantedAt().AsTime(),
	})
	return err //nolint:wrapcheck // the use case named it.
}
