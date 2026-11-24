package keep_flag_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

type In struct {
	Flag string
}

const timeout = 5 * time.Second

func New(db cpatomic.Transactioner, flags Flags) *UseCase {
	return &UseCase{db: db, flags: flags}
}

type UseCase struct {
	db    cpatomic.Transactioner
	flags Flags
}

// Execute keeps the flag for good. One deleted or never opened is no error: nobody can play for it.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id, err := catalog.ParseFlagID(in.Flag)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		flag, err := u.flags.Flag(ctx, id)
		if errors.Is(err, catalog.ErrUnknownFlag) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read the flag: %w", err)
		}
		played, changed := flag.Played()
		if !changed {
			return nil
		}
		if err := u.flags.Save(ctx, played); err != nil {
			return fmt.Errorf("failed to keep the flag: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to keep the flag %s: %w", id, err)
	}
	return nil
}
