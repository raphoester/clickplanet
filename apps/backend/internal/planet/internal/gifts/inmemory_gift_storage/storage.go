//go:build testing

package inmemory_gift_storage

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type given struct {
	tag    tempo.GiftTag
	holder bonuses.Holder
}

func New() *Storage {
	return &Storage{given: cpcolls.NewSet[given]()}
}

type Storage struct {
	mu    sync.Mutex
	given *cpcolls.Set[given]
	err   error
}

var _ gifts.Storage = (*Storage)(nil)

func (s *Storage) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.err = err
}

func (s *Storage) Give(_ context.Context, tag tempo.GiftTag, holder bonuses.Holder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}

	if s.given.Contains(given{tag: tag, holder: holder}) {
		return gifts.ErrGiven
	}

	s.given.Add(given{tag: tag, holder: holder})
	return nil
}

func (s *Storage) Given(tag tempo.GiftTag) []bonuses.Holder {
	s.mu.Lock()
	defer s.mu.Unlock()

	var holders []bonuses.Holder
	s.given.ForEach(func(gift given) {
		if gift.tag == tag {
			holders = append(holders, gift.holder)
		}
	})
	return holders
}
