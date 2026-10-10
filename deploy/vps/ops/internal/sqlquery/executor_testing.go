//go:build testing

package sqlquery

import "context"

type StubExecutor struct {
	Result Result
	Err    error
	Asked  []Query
}

func (s *StubExecutor) Execute(_ context.Context, query Query) (Result, error) {
	s.Asked = append(s.Asked, query)
	return s.Result, s.Err
}
