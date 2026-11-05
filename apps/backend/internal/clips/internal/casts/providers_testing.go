//go:build testing

package casts

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"
)

type ScriptedLedger struct {
	Asked []Window
	takes []Take
	err   error
}

func NewScriptedLedger(takes ...Take) *ScriptedLedger {
	return &ScriptedLedger{takes: takes}
}

func (l *ScriptedLedger) FailWith(err error) {
	l.err = err
}

func (l *ScriptedLedger) Takes(_ context.Context, window Window) ([]Take, error) {
	l.Asked = append(l.Asked, window)
	return l.takes, l.err
}

// The players who let the clips show their names, and nobody else.
type ScriptedNames struct {
	Asked [][]AccountID
	names map[AccountID]string
}

func NewScriptedNames(names map[AccountID]string) *ScriptedNames {
	return &ScriptedNames{names: names}
}

func (n *ScriptedNames) Shown(_ context.Context, accounts []AccountID) (map[AccountID]string, error) {
	n.Asked = append(n.Asked, accounts)
	shown := map[AccountID]string{}
	for _, account := range accounts {
		if name, in := n.names[account]; in {
			shown[account] = name
		}
	}
	return shown, nil
}

type SequentialIDs struct {
	next byte
}

var _ IDProvider = (*SequentialIDs)(nil)

func (s *SequentialIDs) NewID() (CastID, error) {
	s.next++
	var id CastID
	id[0], id[15] = 0xca, s.next
	return id, nil
}

type MemoryStore struct {
	mu    sync.Mutex
	casts map[CastID]CastSnapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{casts: map[CastID]CastSnapshot{}}
}

func (s *MemoryStore) Keep(_ context.Context, cast Cast) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := cast.TakeSnapshot()
	s.casts[kept.ID] = kept
	return nil
}

func (s *MemoryStore) Cast(_ context.Context, id CastID) (Cast, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept, in := s.casts[id]
	if !in {
		return Cast{}, ErrNoCast
	}
	return kept.Restore(), nil
}

func (s *MemoryStore) MadeBefore(_ context.Context, at time.Time) ([]CastID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []CastID
	for _, id := range slices.SortedFunc(maps.Keys(s.casts), compareIDs) {
		if s.casts[id].MadeAt.Before(at) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *MemoryStore) Forget(_ context.Context, id CastID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.casts, id)
	return nil
}

func (s *MemoryStore) ForgetAccount(_ context.Context, account AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, kept := range s.casts {
		for i, fighter := range kept.Fighters {
			if fighter.Account == account {
				kept.Fighters[i].Account, kept.Fighters[i].Name = AccountID{}, ""
			}
		}
		s.casts[id] = kept
	}
	return nil
}

func compareIDs(a, b CastID) int {
	return slices.Compare(a[:], b[:])
}
