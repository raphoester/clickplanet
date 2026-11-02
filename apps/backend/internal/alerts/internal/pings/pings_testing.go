//go:build testing

package pings

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type InMemoryStore struct {
	mu      sync.Mutex
	attacks map[nations.Flag][]time.Time
	starts  map[StartKey]time.Time
}

var _ Store = (*InMemoryStore)(nil)

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{attacks: map[nations.Flag][]time.Time{}, starts: map[StartKey]time.Time{}}
}

func (s *InMemoryStore) AttackPingsOf(_ context.Context, victim nations.Flag, since time.Time) (History, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var history History
	for _, at := range s.attacks[victim] {
		if !at.Before(since) {
			history = append(history, at)
		}
	}
	return history, nil
}

func (s *InMemoryStore) SaveAttackPing(ctx context.Context, victim nations.Flag, at time.Time) error {
	cpatomic.OnCommit(ctx, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.attacks[victim] = append(s.attacks[victim], at)
	})
	return nil
}

func (s *InMemoryStore) StartPinged(_ context.Context, key StartKey) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, found := s.starts[key]
	return found, nil
}

func (s *InMemoryStore) SaveStartPing(ctx context.Context, key StartKey, at time.Time) error {
	cpatomic.OnCommit(ctx, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.starts[key] = at
	})
	return nil
}

type Post struct {
	Channel discord.ChannelID
	Text    string
	Mention discord.RoleID
}

type RecordedPoster struct {
	mu    sync.Mutex
	Posts []Post
	Err   error
}

func (p *RecordedPoster) Post(_ context.Context, channel discord.ChannelID, text string, mention discord.RoleID) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Err != nil {
		return p.Err
	}
	p.Posts = append(p.Posts, Post{Channel: channel, Text: text, Mention: mention})
	return nil
}
