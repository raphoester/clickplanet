package admit_caller_handler

import (
	"context"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Refusals interface {
	Decide(ctx context.Context, caller refusals.Caller, decision refusals.Decision) error
}

func New(refusals Refusals) Handler {
	return Handler{refusals: refusals}
}

type Handler struct {
	refusals Refusals
}

var _ cpredis.Handler[*planetv1.AdmitCaller] = Handler{}

func (h Handler) Handle(ctx context.Context, command *planetv1.AdmitCaller) error {
	caller, err := refusals.ParseCaller(command.GetScope(), command.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the caller is what the command names: the sentinel already says what is wrong.
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return h.refusals.Decide(ctx, caller, refusals.Admitted(command.GetDecidedAt().AsTime())) //nolint:wrapcheck // the book named it.
}
