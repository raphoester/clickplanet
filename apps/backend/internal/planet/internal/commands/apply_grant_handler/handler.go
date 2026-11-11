package apply_grant_handler

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/apply_grant_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

var ErrUnknownKind = errors.New("not a kind of charge")

type UseCase interface {
	Execute(ctx context.Context, in apply_grant_usecase.In) error
}

func New(useCase UseCase) Handler {
	return Handler{useCase: useCase}
}

type Handler struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.GrantCharges] = Handler{}

func (h Handler) Handle(ctx context.Context, command *planetv1.GrantCharges) error {
	grant, err := charges.GrantIDOf(command.GetGrantId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel says what is wrong.
	}
	kind, err := kindOf(command.GetKind())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return h.useCase.Execute(ctx, apply_grant_usecase.In{ //nolint:wrapcheck // the use case named it.
		Grant: grant, Account: command.GetAccountId(), Kind: kind, Amount: int(command.GetAmount()),
	})
}

func kindOf(kind planetv1.BonusKind) (charges.Kind, error) {
	switch kind {
	case planetv1.BonusKind_BONUS_KIND_REFILL:
		return charges.KindRefill, nil
	case planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS:
		return charges.KindSpreadClicks, nil
	case planetv1.BonusKind_BONUS_KIND_BOMB:
		return charges.KindBomb, nil
	case planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS:
		return charges.KindEncloseClicks, nil
	case planetv1.BonusKind_BONUS_KIND_SHIELDS:
		return charges.KindShields, nil
	case planetv1.BonusKind_BONUS_KIND_UNSPECIFIED:
	}

	return "", fmt.Errorf("%w: %v", ErrUnknownKind, kind)
}
