package keep_background_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Files interface {
	Held(ctx context.Context, id backgrounds.BackgroundID) (backgrounds.File, error)
}

type Storage interface {
	Keep(ctx context.Context, background backgrounds.Background) error
}

type In struct {
	ID   backgrounds.BackgroundID
	Name string
}

const readTimeout = 10 * time.Second

func New(files Files, storage Storage, rules backgrounds.Rules, clock cptime.Clock) *UseCase {
	return &UseCase{files: files, storage: storage, rules: rules, clock: clock}
}

type UseCase struct {
	files   Files
	storage Storage
	rules   backgrounds.Rules
	clock   cptime.Clock
}

// The type and the size kept are the bucket's, never what the page said: the rules are checked again on them.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	held, err := u.files.Held(ctx, in.ID)
	if err != nil {
		return fmt.Errorf("cannot read the background sent: %w", err)
	}
	background, err := backgrounds.Keep(held, in.Name, u.clock.Now(), u.rules)
	if err != nil {
		return fmt.Errorf("cannot keep the background: %w", err)
	}
	if err := u.storage.Keep(ctx, background); err != nil {
		return fmt.Errorf("failed to keep the background: %w", err)
	}
	return nil
}
