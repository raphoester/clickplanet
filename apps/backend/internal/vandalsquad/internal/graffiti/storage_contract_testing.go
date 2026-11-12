//go:build testing

package graffiti

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage
	// Every graffiti the storage keeps, first flagged first.
	Kept func(storage Storage) []Snapshot

	storage Storage
}

var contractStart = time.Date(2026, 11, 12, 12, 0, 0, 123_456_000, time.UTC)

var (
	contractAna = cpsession.AccountID{15: 1}
	contractBob = cpsession.AccountID{15: 2}
)

func contractID(name string) ID {
	return ID(uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)))
}

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) graffiti(name string, account cpsession.AccountID, flagged time.Time) Snapshot {
	return Snapshot{
		ID: contractID(name), Account: account, Flag: "ro", Kind: Swastika, Tiles: []drawings.TileID{3, 1, 2},
		FlaggedAt: flagged, SeenAt: flagged.Add(time.Minute), Link: "https://clickplanet.lol/play?lat=1.00&lon=2.00&zoom=20.0",
	}
}

func (s *StorageContractSuite) restored(snapshot Snapshot) Graffiti {
	graffiti, err := snapshot.Restore()
	s.Require().NoError(err)
	return graffiti
}

func (s *StorageContractSuite) saved(name string, account cpsession.AccountID, flagged time.Time) Snapshot {
	snapshot := s.graffiti(name, account, flagged)
	s.Require().NoError(s.storage.Save(s.T().Context(), s.restored(snapshot)))
	return snapshot
}

func (s *StorageContractSuite) snapshots(found []Graffiti, err error) []Snapshot {
	s.Require().NoError(err)

	taken := make([]Snapshot, 0, len(found))
	for _, graffiti := range found {
		taken = append(taken, graffiti.TakeSnapshot())
	}
	return taken
}

func (s *StorageContractSuite) same(want, got []Snapshot) {
	s.Require().Len(got, len(want))
	for i := range want {
		s.True(want[i].FlaggedAt.Equal(got[i].FlaggedAt), "flagged at %s, read %s", want[i].FlaggedAt, got[i].FlaggedAt)
		s.True(want[i].SeenAt.Equal(got[i].SeenAt), "seen at %s, read %s", want[i].SeenAt, got[i].SeenAt)
		got[i].FlaggedAt, got[i].SeenAt = want[i].FlaggedAt, want[i].SeenAt
		s.Equal(want[i], got[i])

		_, err := got[i].Restore()
		s.NoError(err, "what a storage keeps restores a graffiti")
	}
}

func (s *StorageContractSuite) TestNothingIsKeptAtFirst() {
	s.Empty(s.Kept(s.storage))
}

func (s *StorageContractSuite) TestASavedGraffitiIsKeptAsItWas() {
	first := s.saved("first", contractAna, contractStart)
	second := s.saved("second", contractBob, contractStart.Add(time.Hour))

	s.same([]Snapshot{first, second}, s.Kept(s.storage))
}

func (s *StorageContractSuite) TestAGraffitiSavedAgainReplacesItself() {
	first := s.saved("first", contractAna, contractStart)
	again := first
	again.Tiles = []drawings.TileID{2, 5}
	again.SeenAt = contractStart.Add(time.Hour)
	again.Link = "https://clickplanet.lol/play?lat=3.00&lon=4.00&zoom=20.0"

	s.Require().NoError(s.storage.Save(s.T().Context(), s.restored(again)))

	s.same([]Snapshot{again}, s.Kept(s.storage))
}

func (s *StorageContractSuite) TestSeveralGraffitiAreSavedAtOnceAndSavingNoneIsNoError() {
	first := s.graffiti("first", contractAna, contractStart)
	second := s.graffiti("second", contractAna, contractStart.Add(time.Minute))

	s.Require().NoError(s.storage.Save(s.T().Context(), s.restored(first), s.restored(second)))
	s.Require().NoError(s.storage.Save(s.T().Context()))

	s.same([]Snapshot{first, second}, s.Kept(s.storage))
}

func (s *StorageContractSuite) TestAnAnonymizedGraffitiIsKeptAsNobodys() {
	first := s.saved("first", contractAna, contractStart)

	s.Require().NoError(s.storage.Save(s.T().Context(), s.restored(first).Anonymized()))

	first.Account = cpsession.NoAccount
	s.same([]Snapshot{first}, s.Kept(s.storage))
}

func (s *StorageContractSuite) TestWhatAnAccountPaintedComesFirstFlaggedFirstAndNothingElse() {
	later := s.saved("later", contractAna, contractStart.Add(time.Hour))
	s.saved("another's", contractBob, contractStart.Add(time.Minute))
	earlier := s.saved("earlier", contractAna, contractStart)

	s.same([]Snapshot{earlier, later}, s.snapshots(s.storage.PaintedBy(s.T().Context(), contractAna)))
	s.Empty(s.snapshots(s.storage.PaintedBy(s.T().Context(), cpsession.AccountID{15: 9})))
}

func (s *StorageContractSuite) TestNobodyPaintedWhatWasAnonymized() {
	first := s.saved("first", contractAna, contractStart)
	s.Require().NoError(s.storage.Save(s.T().Context(), s.restored(first).Anonymized()))

	s.Empty(s.snapshots(s.storage.PaintedBy(s.T().Context(), contractAna)))
	s.Empty(s.snapshots(s.storage.PaintedBy(s.T().Context(), cpsession.NoAccount)))
}
