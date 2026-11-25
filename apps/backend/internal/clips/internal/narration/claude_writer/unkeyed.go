package claude_writer

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/narration"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpanthropic"
)

type Unkeyed struct{}

var _ narration.Writer = Unkeyed{}

func (Unkeyed) Write(context.Context, narration.Model, narration.Request) (narration.Script, posts.Spend, error) {
	return narration.Script{}, posts.Spend{}, cpanthropic.ErrNoKey
}
