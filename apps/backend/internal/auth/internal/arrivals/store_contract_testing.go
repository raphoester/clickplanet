//go:build testing

package arrivals

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	// Every arrival the store keeps, oldest first.
	Kept func(store Store) []Snapshot

	store Store
}

var contractStart = time.Date(2026, 10, 30, 12, 0, 0, 123_456_000, time.UTC)

var (
	contractAda = AccountID{15: 1}
	contractBob = AccountID{15: 2}
)

func contractID(name string) ArrivalID {
	return ArrivalID(uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)))
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) save(snapshots ...Snapshot) {
	saved := make([]Arrival, 0, len(snapshots))
	for _, snapshot := range snapshots {
		arrival, err := snapshot.Restore()
		s.Require().NoError(err)
		saved = append(saved, arrival)
	}
	s.Require().NoError(s.store.Save(s.T().Context(), saved...))
}

func (s *StoreContractSuite) same(want []Snapshot, got []Snapshot) {
	s.Require().Len(got, len(want))
	for i := range want {
		s.True(want[i].At.Equal(got[i].At), "arrived at %s, read %s", want[i].At, got[i].At)
		got[i].At = want[i].At
		s.Equal(want[i], got[i])
	}
}

func snapshotsOf(found []Arrival) []Snapshot {
	snapshots := make([]Snapshot, 0, len(found))
	for _, arrival := range found {
		snapshots = append(snapshots, arrival.TakeSnapshot())
	}
	return snapshots
}

func (s *StoreContractSuite) TestNothingIsKeptAtFirst() {
	s.Empty(s.Kept(s.store))
}

func (s *StoreContractSuite) TestEverySourceIsKeptAsItWas() {
	recruit := Snapshot{ID: contractID("recruit"), Account: contractAda, Source: SourceRecruit, Referrer: "discord.com", At: contractStart}
	youtube := Snapshot{
		ID: contractID("youtube"), Account: contractBob, Source: SourceYouTube, At: contractStart.Add(time.Second),
	}
	unknown := Snapshot{ID: contractID("unknown"), Account: AccountID{15: 3}, Source: SourceUnknown, At: contractStart.Add(2 * time.Second)}

	s.save(recruit, youtube, unknown)

	s.same([]Snapshot{recruit, youtube, unknown}, s.Kept(s.store))
}

func (s *StoreContractSuite) TestAnArrivalSavedAgainReplacesItself() {
	arrival := Snapshot{ID: contractID("once"), Account: contractAda, Source: SourceRecruit, Referrer: "www.google.com", At: contractStart}
	s.save(arrival)

	anonymized := arrival
	anonymized.Account = AccountID{}
	s.save(anonymized)

	s.same([]Snapshot{anonymized}, s.Kept(s.store))
}

func (s *StoreContractSuite) TestTheArrivalsOfAnAccountAreFoundOldestFirst() {
	later := Snapshot{ID: contractID("later"), Account: contractAda, Source: SourceUnknown, At: contractStart.Add(time.Hour)}
	first := Snapshot{ID: contractID("first"), Account: contractAda, Source: SourceYouTube, At: contractStart}
	other := Snapshot{ID: contractID("other"), Account: contractBob, Source: SourceUnknown, At: contractStart}
	s.save(later, first, other)

	found, err := s.store.Of(s.T().Context(), contractAda)

	s.Require().NoError(err)
	s.same([]Snapshot{first, later}, snapshotsOf(found))
}

func (s *StoreContractSuite) TestAnAnonymizedArrivalNamesNobody() {
	s.save(Snapshot{ID: contractID("gone"), Source: SourceRecruit, Referrer: "www.reddit.com", At: contractStart})

	found, err := s.store.Of(s.T().Context(), AccountID{})

	s.Require().NoError(err)
	s.Empty(found)
}
