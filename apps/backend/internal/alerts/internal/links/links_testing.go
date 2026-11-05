//go:build testing

package links

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

type InMemoryStore struct {
	mu    sync.Mutex
	links map[AccountID]LinkSnapshot
}

var _ Store = (*InMemoryStore)(nil)

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{links: map[AccountID]LinkSnapshot{}}
}

func (s *InMemoryStore) LinkOf(_ context.Context, account AccountID) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept, found := s.links[account]
	if !found {
		return Link{}, ErrNoLink
	}
	return kept.Restore(), nil
}

func (s *InMemoryStore) LinkOfUser(_ context.Context, user discord.UserID) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, kept := range s.links {
		if kept.User == user {
			return kept.Restore(), nil
		}
	}
	return Link{}, ErrNoLink
}

func (s *InMemoryStore) SaveLinks(_ context.Context, saved ...Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, link := range saved {
		kept := link.TakeSnapshot()
		if kept.Forgotten {
			delete(s.links, kept.Account)
			continue
		}
		if earlier, found := s.links[kept.Account]; found && earlier.User == kept.User {
			kept.Name = earlier.Name
		}
		s.links[kept.Account] = kept
	}
	return nil
}

func (s *InMemoryStore) SaveName(_ context.Context, link Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	named := link.TakeSnapshot()
	kept, found := s.links[named.Account]
	if !found || kept.User != named.User {
		return nil
	}
	kept.Name = named.Name
	s.links[named.Account] = kept
	return nil
}

func (s *InMemoryStore) Links() []LinkSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]LinkSnapshot, 0, len(s.links))
	for _, link := range s.links {
		kept = append(kept, link)
	}
	return kept
}

// Player's names, by account: an account it was not told has none.
type FakeNames struct {
	mu    sync.Mutex
	names map[AccountID]Name
	err   error
}

var _ Names = (*FakeNames)(nil)

func NewFakeNames() *FakeNames {
	return &FakeNames{names: map[AccountID]Name{}}
}

func (n *FakeNames) Name(account AccountID, name Name) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.names[account] = name
}

func (n *FakeNames) FailWith(err error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.err = err
}

func (n *FakeNames) NameOf(_ context.Context, account AccountID) (Name, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.err != nil {
		return "", n.err
	}
	return n.names[account], nil
}

// The nicknames the bot gave, in order, as "user name"; a user listed in Gone left the server, and Refused the bot may
// not rename.
type FakeRenamer struct {
	mu      sync.Mutex
	renamed []string
	Gone    map[discord.UserID]bool
	Refused map[discord.UserID]bool
	Err     error
}

var _ Renamer = (*FakeRenamer)(nil)

func (r *FakeRenamer) Rename(_ context.Context, user discord.UserID, nick string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case r.Err != nil:
		return r.Err
	case r.Gone[user]:
		return discord.ErrNotMember
	case r.Refused[user]:
		return discord.ErrRefused
	}
	r.renamed = append(r.renamed, string(user)+" "+nick)
	return nil
}

func (r *FakeRenamer) Renamed() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.renamed...)
}
