//go:build testing

package verdicts

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu   sync.Mutex
	kept map[AccountID]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{kept: map[AccountID]Snapshot{}}
}

func (s *MemoryStore) Keep(_ context.Context, verdicts []Verdict) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, verdict := range verdicts {
		kept := verdict.TakeSnapshot()
		s.kept[kept.Account] = kept
	}
	return nil
}

func (s *MemoryStore) Of(_ context.Context, accounts []AccountID) ([]Verdict, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var found []Verdict
	for _, account := range accounts {
		if kept, in := s.kept[account]; in {
			found = append(found, kept.Restore())
		}
	}
	return found, nil
}

func (s *MemoryStore) Verdict(_ context.Context, account AccountID) (Verdict, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept, in := s.kept[account]
	if !in {
		return Verdict{}, ErrNoVerdict
	}
	return kept.Restore(), nil
}

func (s *MemoryStore) Forget(_ context.Context, account AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.kept, account)
	return nil
}

// A judge that answers each name it was told, and leaves the others out.
type ScriptedJudge struct {
	Asked   [][]string
	answers map[string]Judgement
	err     error
}

func NewScriptedJudge(answers ...Judgement) *ScriptedJudge {
	byName := map[string]Judgement{}
	for _, answer := range answers {
		byName[answer.Name] = answer
	}
	return &ScriptedJudge{answers: byName}
}

func (j *ScriptedJudge) FailWith(err error) {
	j.err = err
}

func (j *ScriptedJudge) Judge(_ context.Context, names []string) ([]Judgement, error) {
	j.Asked = append(j.Asked, names)
	if j.err != nil {
		return nil, j.err
	}
	var answered []Judgement
	for _, name := range names {
		if answer, in := j.answers[name]; in {
			answered = append(answered, answer)
		}
	}
	return answered, nil
}
