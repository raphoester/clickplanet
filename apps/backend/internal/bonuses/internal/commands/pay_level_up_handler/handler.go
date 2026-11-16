package pay_level_up_handler

import (
	"context"
	"errors"
	"fmt"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/planetwords"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

var ErrUnknownKind = errors.New("not a kind the bonuses module draws")

type UseCase interface {
	Execute(ctx context.Context, levelUp bonuses.LevelUp) error
}

func New(useCase UseCase) Handler {
	return Handler{useCase: useCase}
}

// Handler pays a level-up progression told, in the bonuses module's words.
type Handler struct {
	useCase UseCase
}

var _ cpredis.Handler[*bonusesv1.PayLevelUp] = Handler{}

func (h Handler) Handle(ctx context.Context, command *bonusesv1.PayLevelUp) error {
	account, err := bonuses.ParseAccount(command.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel says what is wrong.
	}
	grant, err := bonuses.GrantIDOf(command.GetGrantId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel says what is wrong.
	}
	unlocked := make(bonuses.Unlocked, 0, len(command.GetUnlocked()))
	for _, theirs := range command.GetUnlocked() {
		kind, ok := planetwords.KindOf(theirs)
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnknownKind, theirs)
		}
		unlocked = append(unlocked, kind)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return h.useCase.Execute(ctx, bonuses.LevelUp{Account: account, Grant: grant, Unlocked: unlocked}) //nolint:wrapcheck // the use case named it.
}
