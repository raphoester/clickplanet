package claude_editor

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/usecases/choose_clips_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

// The editor of a job with no ANTHROPIC_API_KEY: it sends nothing, and every run drawn for the model falls back to
// the rules, saying why.
type Unkeyed struct{}

var _ choose_clips_usecase.Editor = Unkeyed{}

func (Unkeyed) Edit(context.Context, posts.Offer) (posts.Edition, posts.Spend, error) {
	return posts.Edition{}, posts.Spend{}, cpanthropic.ErrNoKey
}

func (Unkeyed) Count(context.Context, posts.Offer) (int64, error) {
	return 0, cpanthropic.ErrNoKey
}
