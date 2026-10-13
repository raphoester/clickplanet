package choose_clips_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

type Shortlists interface {
	Shortlist(ctx context.Context) (posts.Shortlist, error)
}

type Executor interface {
	Execute(ctx context.Context) (posts.Shortlist, error)
}

func New(shortlists Shortlists, perRun int) *UseCase {
	return &UseCase{shortlists: shortlists, perRun: perRun}
}

type UseCase struct {
	shortlists Shortlists
	perRun     int
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (posts.Shortlist, error) {
	shortlist, err := u.shortlists.Shortlist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the stories worth a clip: %w", err)
	}
	return shortlist.Best(u.perRun), nil
}
