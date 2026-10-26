package play_rule_set_handler

import (
	"context"
	"errors"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Rules interface {
	Play(ctx context.Context, choice rules.Choice) (rules.Snapshot, bool, error)
}

func New(rules Rules) Handler {
	return Handler{rules: rules}
}

type Handler struct {
	rules Rules
}

var _ cpredis.Handler[*planetv1.PlayRuleSet] = Handler{}

func (h Handler) Handle(ctx context.Context, command *planetv1.PlayRuleSet) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	_, _, err := h.rules.Play(ctx, rules.Choice{
		Set: rules.SetName(command.GetRuleSet()), DecidedAt: command.GetDecidedAt().AsTime(),
	})
	// A set refused is logged and dropped: delivered again, it would hold back the next choice.
	if errors.Is(err, rules.ErrUnknownSet) || errors.Is(err, rules.ErrInvalidSet) {
		return nil
	}
	return err //nolint:wrapcheck // the use case named it.
}
