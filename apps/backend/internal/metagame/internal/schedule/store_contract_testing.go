//go:build testing

package schedule

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var finaleStarts = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func kept(n byte, startsAt, endsAt time.Time) Entry {
	return EntrySnapshot{
		ID: EntryID(uuid.UUID{15: n}), Set: "rush", Kind: Battle, StartsAt: startsAt, EndsAt: endsAt, Priority: Priority(n),
		MadeAt: finaleStarts.Add(-time.Duration(n) * time.Hour).Add(123456 * time.Microsecond),
	}.Restore()
}

func (s *StoreContractSuite) save(entries ...Entry) {
	for _, entry := range entries {
		s.Require().NoError(s.store.Save(s.T().Context(), entry))
	}
}

func (s *StoreContractSuite) snapshots(entries []Entry) []EntrySnapshot {
	snapshots := make([]EntrySnapshot, 0, len(entries))
	for _, entry := range entries {
		snapshot := entry.TakeSnapshot()
		snapshot.StartsAt, snapshot.EndsAt, snapshot.MadeAt = snapshot.StartsAt.UTC(), snapshot.EndsAt.UTC(), snapshot.MadeAt.UTC()
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

func (s *StoreContractSuite) TestAnEntryIsReadAsItWasKept() {
	entry := kept(1, finaleStarts, finaleStarts.Add(2*time.Hour))
	s.save(entry)

	read, err := s.store.Entry(s.T().Context(), entry.id)
	s.Require().NoError(err)
	s.Equal(s.snapshots([]Entry{entry}), s.snapshots([]Entry{read}))
}

func (s *StoreContractSuite) TestAnEntryNeverKeptIsUnknown() {
	_, err := s.store.Entry(s.T().Context(), EntryID(uuid.UUID{15: 9}))
	s.Require().ErrorIs(err, ErrUnknownEntry)
}

func (s *StoreContractSuite) TestAnEntryKeptAgainReplacesItsValues() {
	entry := kept(1, finaleStarts, finaleStarts.Add(2*time.Hour))
	s.save(entry)
	snapshot := entry.TakeSnapshot()
	snapshot.Set, snapshot.Kind, snapshot.Priority, snapshot.EndsAt = "calm", Plain, 4, finaleStarts.Add(time.Hour)
	s.save(snapshot.Restore())

	read, err := s.store.Entry(s.T().Context(), entry.id)
	s.Require().NoError(err)
	s.Equal(s.snapshots([]Entry{snapshot.Restore()}), s.snapshots([]Entry{read}))
}

func (s *StoreContractSuite) TestTheUnendedEntriesAreTheOnesThatEndAfterATime() {
	ended := kept(1, finaleStarts.Add(-2*time.Hour), finaleStarts)
	running := kept(2, finaleStarts.Add(-time.Hour), finaleStarts.Add(time.Nanosecond*1000))
	coming := kept(3, finaleStarts.Add(time.Hour), finaleStarts.Add(2*time.Hour))
	s.save(ended, running, coming)

	unended, err := s.store.Unended(s.T().Context(), finaleStarts)
	s.Require().NoError(err)
	s.ElementsMatch(s.snapshots([]Entry{running, coming}), s.snapshots(unended), "an entry is over at its end")
}

func (s *StoreContractSuite) TestADeletedEntryIsGone() {
	entry, other := kept(1, finaleStarts, finaleStarts.Add(time.Hour)), kept(2, finaleStarts, finaleStarts.Add(time.Hour))
	s.save(entry, other)

	s.Require().NoError(s.store.Delete(s.T().Context(), entry.id))

	_, err := s.store.Entry(s.T().Context(), entry.id)
	s.Require().ErrorIs(err, ErrUnknownEntry)
	unended, err := s.store.Unended(s.T().Context(), time.Time{})
	s.Require().NoError(err)
	s.Equal(s.snapshots([]Entry{other}), s.snapshots(unended))
}
