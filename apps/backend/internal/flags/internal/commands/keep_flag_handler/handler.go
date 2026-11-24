package keep_flag_handler

import (
	"context"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/usecases/keep_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in keep_flag_usecase.In) error
}

func New(useCase UseCase) Handler {
	return Handler{useCase: useCase}
}

// Handler takes planet's word that a player played for the flag.
type Handler struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.KeepFlag] = Handler{}

func (h Handler) Handle(ctx context.Context, command *flagsv1.KeepFlag) error {
	return h.useCase.Execute(ctx, keep_flag_usecase.In{Flag: command.GetFlag()}) //nolint:wrapcheck // the use case named it.
}
