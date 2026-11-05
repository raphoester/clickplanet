package keep_snapshot_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/snapshots"
)

func New(storage snapshots.Storage) *UseCase {
	return &UseCase{storage: storage}
}

type UseCase struct {
	storage snapshots.Storage
}

// The rows of the part kept.
func (u *UseCase) Execute(ctx context.Context, part snapshots.Part) (int, error) {
	if err := u.storage.Keep(ctx, part); err != nil {
		return 0, fmt.Errorf("failed to keep a part of a snapshot: %w", err)
	}
	return len(part.TakeSnapshot().Rows), nil
}
