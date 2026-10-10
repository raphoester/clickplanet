package accesslog

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

type Query struct {
	Since    time.Time
	Until    time.Time
	Contains string
	Limit    excerpt.Limit
}

type Reader interface {
	Read(ctx context.Context, query Query) (excerpt.Excerpt, error)
}
