//go:build testing

package clicks

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"
)

// AllegianceStorageContractSuite is the behaviour every AllegianceStorage shares. Embed it and set NewStorage.
type AllegianceStorageContractSuite struct {
	suite.Suite

	// NewStorage builds an empty store.
	NewStorage func() AllegianceStorage

	storage AllegianceStorage
}

var (
	contractEpoch = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	contractAda   = AccountAllegianceKey("ada")
	contractHome  = ScopeAllegianceKey("2001:db8::/64")
)

func (s *AllegianceStorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *AllegianceStorageContractSuite) save(tallies map[AllegianceKey]Allegiance) {
	s.Require().NoError(s.storage.SaveAllegiances(context.Background(), tallies))
}

func (s *AllegianceStorageContractSuite) read(keys ...AllegianceKey) map[AllegianceKey]Allegiance {
	tallies, err := s.storage.Allegiances(context.Background(), keys...)
	s.Require().NoError(err)

	return tallies
}

func (s *AllegianceStorageContractSuite) TestAnEmptyStoreHasNoTally() {
	s.Empty(s.read(contractAda, contractHome))
}

func (s *AllegianceStorageContractSuite) TestATallyReadsBackAsItWasSaved() {
	tally := Allegiance{}.With("pl", contractEpoch).With("pl", contractEpoch.Add(time.Hour)).With("ad", contractEpoch.Add(time.Hour))

	s.save(map[AllegianceKey]Allegiance{contractAda: tally})

	s.Equal(map[AllegianceKey]Allegiance{contractAda: tally}, s.read(contractAda, contractHome), "a key with no tally is absent")
}

func (s *AllegianceStorageContractSuite) TestASaveReplacesATallyAndLeavesTheOthers() {
	first := Allegiance{}.With("pl", contractEpoch)
	second := first.With("ad", contractEpoch.Add(time.Hour))
	s.save(map[AllegianceKey]Allegiance{contractAda: first, contractHome: first})

	s.save(map[AllegianceKey]Allegiance{contractAda: second})

	s.Equal(map[AllegianceKey]Allegiance{contractAda: second, contractHome: first}, s.read(contractAda, contractHome))
}

func (s *AllegianceStorageContractSuite) TestSavingNothingChangesNothing() {
	s.save(map[AllegianceKey]Allegiance{})

	s.Empty(s.read(contractAda))
}

func (s *AllegianceStorageContractSuite) TestReadingNoKeyAnswersNoTally() {
	s.save(map[AllegianceKey]Allegiance{contractAda: Allegiance{}.With("pl", contractEpoch)})

	s.Empty(s.read())
}

func (s *AllegianceStorageContractSuite) TestATallyComesBackAtTheSameInstant() {
	at := time.Date(2026, 10, 1, 15, 4, 5, 123_000_000, time.FixedZone("Sofia", 3*60*60))
	s.save(map[AllegianceKey]Allegiance{contractAda: Allegiance{}.With("bg", at)})

	back := s.read(contractAda)[contractAda]

	s.True(back.At().Equal(at), "%v is not %v", back.At(), at)
	s.Equal("bg", back.Flag())
}

func (s *AllegianceStorageContractSuite) TestTalliesWithNoTakeSinceTheCutoffAreDeleted() {
	s.save(map[AllegianceKey]Allegiance{
		contractAda:  Allegiance{}.With("pl", contractEpoch),
		contractHome: Allegiance{}.With("pl", contractEpoch.Add(time.Hour)),
	})

	deleted, err := s.storage.DeleteAllegiancesBefore(context.Background(), contractEpoch.Add(time.Minute))
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	s.Equal([]AllegianceKey{contractHome}, keysOf(s.read(contractAda, contractHome)))
}

func (s *AllegianceStorageContractSuite) TestATallyTakenAtTheCutoffIsKept() {
	s.save(map[AllegianceKey]Allegiance{contractAda: Allegiance{}.With("pl", contractEpoch)})

	deleted, err := s.storage.DeleteAllegiancesBefore(context.Background(), contractEpoch)
	s.Require().NoError(err)

	s.Zero(deleted)
	s.Len(s.read(contractAda), 1)
}

func keysOf(tallies map[AllegianceKey]Allegiance) []AllegianceKey {
	keys := make([]AllegianceKey, 0, len(tallies))
	for key := range tallies {
		keys = append(keys, key)
	}

	return keys
}
