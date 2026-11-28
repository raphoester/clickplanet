package set_anthem_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrNoAccount = errors.New("no account named the caller")

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

type In struct {
	Account string
	Flag    string
	// The library code of the anthem; empty takes it off.
	Anthem string
}

const timeout = 5 * time.Second

func New(clock cptime.Clock, db cpatomic.Transactioner, flags Flags, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{clock: clock, db: db, flags: flags, outbox: outbox}
}

type UseCase struct {
	clock  cptime.Clock
	db     cpatomic.Transactioner
	flags  Flags
	outbox cpatomic.Outbox
}

// The holder's anthem, kept with AnthemChanged in one transaction, the flag read for update: a payment that claims it
// at once is never undone. The same anthem again changes and tells nothing.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	account, err := uuid.Parse(in.Account)
	if err != nil || account == uuid.Nil {
		return fmt.Errorf("%w: %q", ErrNoAccount, in.Account)
	}
	id, err := catalog.ParseFlagID(in.Flag)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	anthem, err := catalog.ParseAnthem(in.Anthem)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}

	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		flag, err := u.flags.Flag(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the flag: %w", err)
		}
		sung, fact, err := flag.Sung(catalog.AccountID(account), anthem, u.clock.Now())
		if err != nil || fact == nil {
			return err //nolint:wrapcheck // the rule names it.
		}
		if err := u.flags.Save(ctx, sung); err != nil {
			return fmt.Errorf("failed to keep the flag: %w", err)
		}
		if err := u.outbox.Record(ctx, fact); err != nil {
			return fmt.Errorf("failed to record that the flag's anthem changed: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to set the anthem of %s: %w", id, err)
	}
	return nil
}
