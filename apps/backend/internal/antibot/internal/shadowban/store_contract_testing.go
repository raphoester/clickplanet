//go:build testing

package shadowban

import (
	"sync"
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	Key      func(n int) string

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 10, 8, 12, 30, 0, 123456000, time.UTC)

func (s *StoreContractSuite) keep(record Record) {
	s.Require().NoError(s.store.Change(s.T().Context(), record.Key, func(Record, bool) (Record, bool) {
		return record, true
	}))
}

func (s *StoreContractSuite) record(key string) (Record, bool) {
	record, found, err := s.store.Record(s.T().Context(), key)
	s.Require().NoError(err)
	return record, found
}

func (s *StoreContractSuite) TestAKeyNeverChangedIsNotFound() {
	_, found := s.record(s.Key(1))
	s.False(found)
}

func (s *StoreContractSuite) TestAChangeIsKeptAndReadBack() {
	flagged := Record{Key: s.Key(1), Flags: 3, Offences: 2, ExpiresAt: contractAt, LastFlaggedAt: contractAt.Add(-time.Hour)}
	banned := Record{Key: s.Key(2), Flags: 0, Offences: 1, ExpiresAt: contractAt.Add(time.Hour)}
	s.keep(flagged)
	s.keep(banned)

	record, found := s.record(s.Key(1))
	s.Require().True(found)
	s.Equal(flagged, record)

	record, found = s.record(s.Key(2))
	s.Require().True(found)
	s.Equal(banned, record, "a ban that was never flagged has no flag time")
}

func (s *StoreContractSuite) TestAChangeSeesWhatWasKeptAndReplacesIt() {
	s.keep(Record{Key: s.Key(1), Flags: 1, Offences: 1, ExpiresAt: contractAt})

	s.Require().NoError(s.store.Change(s.T().Context(), s.Key(1), func(record Record, found bool) (Record, bool) {
		s.True(found)
		s.Equal(s.Key(1), record.Key)
		record.Flags++
		record.Offences = 0
		record.ExpiresAt = contractAt.Add(-time.Minute)
		return record, true
	}))

	record, _ := s.record(s.Key(1))
	s.Equal(Record{Key: s.Key(1), Flags: 2, ExpiresAt: contractAt.Add(-time.Minute)}, record)
}

func (s *StoreContractSuite) TestAChangeOfAKeyNeverKeptStartsFromNothing() {
	s.Require().NoError(s.store.Change(s.T().Context(), s.Key(1), func(record Record, found bool) (Record, bool) {
		s.False(found)
		s.Equal(Record{Key: s.Key(1)}, record)
		return record, false
	}))
}

func (s *StoreContractSuite) TestAChangeThatKeepsNothingWritesNothing() {
	s.keep(Record{Key: s.Key(1), Flags: 1, Offences: 1, ExpiresAt: contractAt})

	for _, key := range []string{s.Key(1), s.Key(2)} {
		s.Require().NoError(s.store.Change(s.T().Context(), key, func(Record, bool) (Record, bool) {
			return Record{Key: key, Flags: 9, Offences: 9, ExpiresAt: contractAt.Add(time.Hour)}, false
		}))
	}

	record, _ := s.record(s.Key(1))
	s.Equal(1, record.Flags)
	_, found := s.record(s.Key(2))
	s.False(found)
}

func (s *StoreContractSuite) TestChangesToOneKeyNeverOverlap() {
	const changes = 20

	var wg sync.WaitGroup
	for range changes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.NoError(s.store.Change(s.T().Context(), s.Key(1), func(record Record, _ bool) (Record, bool) {
				record.Flags++
				return record, true
			}))
		}()
	}
	wg.Wait()

	record, _ := s.record(s.Key(1))
	s.Equal(changes, record.Flags, "each change saw the one before it")
}

func (s *StoreContractSuite) TestRunningCountsTheBansThatEndAfterNow() {
	s.keep(Record{Key: s.Key(1), Offences: 1, ExpiresAt: contractAt.Add(time.Second)})
	s.keep(Record{Key: s.Key(2), Offences: 1, ExpiresAt: contractAt})
	s.keep(Record{Key: s.Key(3), Offences: 2, ExpiresAt: contractAt.Add(-time.Hour)})

	running, err := s.store.Running(s.T().Context(), contractAt)
	s.Require().NoError(err)
	s.Equal(1, running, "a ban that ends now has ended")
}
