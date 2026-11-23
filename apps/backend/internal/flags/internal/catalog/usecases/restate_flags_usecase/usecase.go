package restate_flags_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Flags interface {
	Flags(ctx context.Context) ([]catalog.Flag, error)
}

const timeout = 30 * time.Second

func New(db cpatomic.Transactioner, flags Flags, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{db: db, flags: flags, outbox: outbox}
}

// UseCase tells every flag again as it is now, for a module that subscribed after it was told.
type UseCase struct {
	db     cpatomic.Transactioner
	flags  Flags
	outbox cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		flags, err := u.flags.Flags(ctx)
		if err != nil {
			return fmt.Errorf("failed to read the flags: %w", err)
		}
		var facts []proto.Message
		for _, flag := range flags {
			facts = append(facts, flag.Facts()...)
		}
		if len(facts) == 0 {
			return nil
		}
		if err := u.outbox.Record(ctx, facts...); err != nil {
			return fmt.Errorf("failed to record every flag again: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to tell every flag again: %w", err)
	}
	return nil
}
