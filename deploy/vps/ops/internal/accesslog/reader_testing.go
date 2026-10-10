//go:build testing

package accesslog

import (
	"context"

	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

type StubReader struct {
	Excerpt excerpt.Excerpt
	Err     error
	Asked   []Query
}

func (s *StubReader) Read(_ context.Context, query Query) (excerpt.Excerpt, error) {
	s.Asked = append(s.Asked, query)
	return s.Excerpt, s.Err
}
