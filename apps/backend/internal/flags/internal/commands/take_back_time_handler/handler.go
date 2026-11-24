package take_back_time_handler

import (
	"context"
	"log/slog"
	"time"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const timeout = 5 * time.Second

type UseCase interface {
	Execute(ctx context.Context, id grants.GrantID) error
}

func New(useCase UseCase, logger *slog.Logger) Handler {
	return Handler{useCase: useCase, logger: logger}
}

// The command's edge: a payment's time, taken back once.
type Handler struct {
	useCase UseCase
	logger  *slog.Logger
}

var _ cpredis.Handler[*flagsv1.TakeBackTime] = Handler{}

func (h Handler) Handle(ctx context.Context, command *flagsv1.TakeBackTime) error {
	id, err := grants.ParseGrantID(command.GetGrantId())
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := h.useCase.Execute(ctx, id); err != nil {
		return err //nolint:wrapcheck // the use case named it.
	}
	h.logger.Info("time paid for taken back", slog.String("grant", id.String()))
	return nil
}
