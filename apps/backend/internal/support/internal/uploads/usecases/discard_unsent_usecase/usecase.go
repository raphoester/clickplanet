package discard_unsent_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads"
)

type Uploads interface {
	Unsent(ctx context.Context, cutoff time.Time, limit int) ([]uploads.Upload, error)
	Save(ctx context.Context, uploads ...uploads.Upload) error
}

type Bucket interface {
	Delete(ctx context.Context, ids ...notes.AttachmentID) error
}

const (
	timeout = 30 * time.Second
	batch   = 100
)

// keepFor is how long a file may wait for the note that names it: past it, it is deleted.
func New(keepFor time.Duration, clock cptime.Clock, kept Uploads, bucket Bucket) *UseCase {
	return &UseCase{keepFor: keepFor, clock: clock, uploads: kept, bucket: bucket}
}

type UseCase struct {
	keepFor time.Duration
	clock   cptime.Clock
	uploads Uploads
	bucket  Bucket
}

func (u *UseCase) Execute(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cutoff := u.clock.Now().Add(-u.keepFor)
	var discarded int64
	for {
		unsent, err := u.uploads.Unsent(ctx, cutoff, batch)
		if err != nil {
			return discarded, fmt.Errorf("failed to read the files no note names: %w", err)
		}
		if len(unsent) == 0 {
			return discarded, nil
		}
		if err := u.bucket.Delete(ctx, uploads.FilesOf(unsent)...); err != nil {
			return discarded, fmt.Errorf("failed to delete the files no note names: %w", err)
		}
		if err := u.uploads.Save(ctx, uploads.AllDiscarded(unsent)...); err != nil {
			return discarded, fmt.Errorf("failed to free the room of the files deleted: %w", err)
		}
		discarded += int64(len(unsent))
	}
}
