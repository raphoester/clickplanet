package suggest_feature_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type Saver interface {
	Save(ctx context.Context, notes ...notes.Note) error
}

type In struct {
	Account notes.AccountID
	Text    string
}

const writeTimeout = 5 * time.Second

func New(ids notes.IDProvider, clock cptime.Clock, saver Saver) *UseCase {
	return &UseCase{ids: ids, clock: clock, saver: saver}
}

type UseCase struct {
	ids   notes.IDProvider
	clock cptime.Clock
	saver Saver
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	id, err := u.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to draw a note id: %w", err)
	}

	note, err := notes.NewSuggestion(id, in.Account, in.Text, u.clock.Now())
	if err != nil {
		return fmt.Errorf("cannot take the suggestion: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := u.saver.Save(ctx, note); err != nil {
		return fmt.Errorf("failed to keep the suggestion: %w", err)
	}
	return nil
}
