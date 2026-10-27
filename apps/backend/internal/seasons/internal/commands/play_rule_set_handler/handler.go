package play_rule_set_handler

import (
	"context"
	"errors"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules/usecases/play_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Rules interface {
	Play(ctx context.Context, choice rules.Choice) (play_rules_usecase.Played, bool, error)
}

func New(rules Rules) Handler {
	return Handler{rules: rules}
}

type Handler struct {
	rules Rules
}

var _ cpredis.Handler[*seasonsv1.PlayRuleSet] = Handler{}

func (h Handler) Handle(ctx context.Context, command *seasonsv1.PlayRuleSet) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	choice := rules.Choice{Set: rules.SetName(command.GetRuleSet()), DecidedAt: command.GetDecidedAt().AsTime()}
	if command.GetEndsAt() != nil {
		choice.Window = rules.Window{StartsAt: command.GetStartsAt().AsTime(), EndsAt: command.GetEndsAt().AsTime()}
	}
	_, _, err := h.rules.Play(ctx, choice)
	// A set refused is logged and dropped: delivered again, it would hold back the next choice.
	if errors.Is(err, rules.ErrUnknownSet) || errors.Is(err, rules.ErrInvalidSet) {
		return nil
	}
	return err //nolint:wrapcheck // the use case named it.
}
