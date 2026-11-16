package keep_unlocks_handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/planetwords"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

var ErrUnknownKind = errors.New("not a kind the bonuses module draws")

type UseCase interface {
	Execute(ctx context.Context, account bonuses.Account, decision unlocks.Decision) error
}

func New(useCase UseCase) Handler {
	return Handler{useCase: useCase}
}

// Handler keeps the kinds progression unlocked for an account, told in the bonuses module's words.
type Handler struct {
	useCase UseCase
}

var _ cpredis.Handler[*bonusesv1.KeepUnlocks] = Handler{}

func (h Handler) Handle(ctx context.Context, command *bonusesv1.KeepUnlocks) error {
	account, err := bonuses.ParseAccount(command.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel says what is wrong.
	}
	kinds := make(bonuses.Unlocked, 0, len(command.GetKinds()))
	for _, theirs := range command.GetKinds() {
		kind, ok := planetwords.KindOf(theirs)
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnknownKind, theirs)
		}
		kinds = append(kinds, kind)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return h.useCase.Execute(ctx, account, unlocks.DecisionOf(kinds, decidedAt(command))) //nolint:wrapcheck // the use case named it.
}

// A command with no time reads as the zero time, which the use case refuses, not as 1970.
func decidedAt(command *bonusesv1.KeepUnlocks) time.Time {
	if command.GetDecidedAt() == nil {
		return time.Time{}
	}
	return command.GetDecidedAt().AsTime()
}
