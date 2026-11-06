package save_rules_usecase

import (
	"context"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/rules"
)

type Store interface {
	Save(ctx context.Context, rules rules.Rules) error
}

func New(store Store) *UseCase {
	return &UseCase{store: store}
}

// UseCase replaces the numbers missions draws with. The next draw reads them.
type UseCase struct {
	store Store
}

func (u *UseCase) Execute(ctx context.Context, in *missionsv1.Rules) error {
	kept, err := rules.RulesOf(in)
	if err != nil {
		return err //nolint:wrapcheck // it names every number that cannot play.
	}
	return u.store.Save(ctx, kept) //nolint:wrapcheck // the store says what it could not keep.
}
