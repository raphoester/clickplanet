package keep_appearances_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

type Appearances interface {
	// How many of its players the server kept as named.
	Keep(ctx context.Context, appearance posts.Appearance) (int, error)
}

type Executor interface {
	Execute(ctx context.Context, report posts.Report) []Kept
}

func New(appearances Appearances) *UseCase {
	return &UseCase{appearances: appearances}
}

type UseCase struct {
	appearances Appearances
}

var _ Executor = (*UseCase)(nil)

// What became of one post's players. A post the server could not keep is not told in the game, and fails nothing: the
// clip is out already.
type Kept struct {
	Appearance posts.Appearance
	Named      int
	Err        error
}

func (u *UseCase) Execute(ctx context.Context, report posts.Report) []Kept {
	named := report.Appearances()
	kept := make([]Kept, 0, len(named))
	for _, appearance := range named {
		count, err := u.appearances.Keep(ctx, appearance)
		kept = append(kept, Kept{Appearance: appearance, Named: count, Err: err})
	}
	return kept
}
