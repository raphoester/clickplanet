package answer_appeal_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func New(db cpatomic.Transactioner, store appeals.Store) *UseCase {
	return &UseCase{db: db, store: store}
}

type UseCase struct {
	db    cpatomic.Transactioner
	store appeals.Store
}

// Execute answers the submission's appeal with the verdict decided at a time: a verdict told again, or one from
// before the appeal, changes nothing.
func (u *UseCase) Execute(ctx context.Context, submission string, decidedAt time.Time) error {
	id, err := submissions.ParseSubmissionID(submission)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	return u.db.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step names its own.
		kept, err := u.store.Of(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the appeals of %s: %w", id, err)
		}
		for _, appeal := range kept {
			answered, changed := appeal.Answered(decidedAt)
			if !changed {
				continue
			}
			if err := u.store.Save(ctx, answered); err != nil {
				return fmt.Errorf("failed to answer the appeal of %s: %w", id, err)
			}
		}
		return nil
	})
}
