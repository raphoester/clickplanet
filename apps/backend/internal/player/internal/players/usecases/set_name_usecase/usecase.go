package set_name_usecase

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
	SaveProfile(ctx context.Context, profile players.Profile) error
}

type Accounts interface {
	Linked(ctx context.Context, account players.AccountID) (bool, error)
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	profiles      Profiles
	accounts      Accounts
	outbox        cpatomic.Outbox
	clock         cptime.Clock
}

func New(
	transactioner cpatomic.Transactioner, profiles Profiles, accounts Accounts, outbox cpatomic.Outbox, clock cptime.Clock,
) *UseCase {
	return &UseCase{transactioner: transactioner, profiles: profiles, accounts: accounts, outbox: outbox, clock: clock}
}

type In struct {
	Account players.AccountID
	Name    string
}

func (u *UseCase) Execute(ctx context.Context, in In) (players.Profile, error) {
	name, err := players.NameOf(in.Name)
	if err != nil {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the domain's sentinel.
	}

	linked, err := u.accounts.Linked(ctx, in.Account)
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to ask whether the account is linked: %w", err)
	}
	if !linked {
		return players.Profile{}, players.ErrNotLinked
	}

	current, err := u.profiles.Profile(ctx, in.Account)
	made := errors.Is(err, players.ErrNoProfile)
	if made {
		current, err = players.Unnamed(in.Account), nil
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	now := u.clock.Now()
	profile, err := current.Renamed(name, now)
	if err != nil {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the domain's refusal.
	}

	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.profiles.SaveProfile(ctx, profile); err != nil {
			return err //nolint:wrapcheck // wrapped below, past the sentinel.
		}
		told := []proto.Message{&playerv1.PlayerNamed{AccountId: in.Account.String()}}
		if made {
			told = append(told, &playerv1.PlayerCreated{AccountId: in.Account.String()})
		}
		// The past names follow from the event: the save does not wait on them (docs/transactions.md).
		if left, ok := pastnames.PastNameOf(current, profile, now); ok {
			kept := left.TakeSnapshot()
			told = append(told, &playerv1.NameLeft{
				AccountId: kept.Account.String(), Name: string(kept.Name), LeftAt: timestamppb.New(kept.LeftAt),
			})
		}
		return u.outbox.Record(ctx, told...) //nolint:wrapcheck // wrapped below.
	})
	if errors.Is(err, players.ErrNameTaken) {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the port's sentinel.
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to save the name: %w", err)
	}

	return profile, nil
}
