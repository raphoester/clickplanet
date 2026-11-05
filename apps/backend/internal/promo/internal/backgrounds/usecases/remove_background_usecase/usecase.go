package remove_background_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
)

type Files interface {
	Delete(ctx context.Context, id backgrounds.BackgroundID) error
}

type Storage interface {
	Remove(ctx context.Context, id backgrounds.BackgroundID) error
}

const writeTimeout = 10 * time.Second

func New(files Files, storage Storage) *UseCase {
	return &UseCase{files: files, storage: storage}
}

type UseCase struct {
	files   Files
	storage Storage
}

// The row goes first, refused while a card shows it, so no card ever shows a file the bucket no longer holds. A file
// the bucket then would not delete stays there, listed nowhere.
func (u *UseCase) Execute(ctx context.Context, id backgrounds.BackgroundID) error {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	if err := u.storage.Remove(ctx, id); err != nil {
		return fmt.Errorf("cannot remove the background: %w", err)
	}
	if err := u.files.Delete(ctx, id); err != nil {
		return fmt.Errorf("removed the background, but not its file: %w", err)
	}
	return nil
}
