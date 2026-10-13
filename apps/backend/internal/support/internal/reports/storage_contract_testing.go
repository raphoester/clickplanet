//go:build testing

package reports

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage
	// Every report the storage keeps, oldest first.
	Kept func(storage Storage) []Snapshot

	storage Storage
}

var contractStart = time.Date(2026, 10, 11, 12, 0, 0, 123_456_000, time.UTC)

var (
	contractAda = AccountID{15: 1}
	contractBob = AccountID{15: 2}
	contractEve = AccountID{15: 3}
)

func contractID(name string) ReportID {
	return ReportID(uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)))
}

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) report(name string, reporter AccountID, reported AccountID, at time.Time) Snapshot {
	return Snapshot{ID: contractID(name), Reporter: reporter, Reported: reported, Complaint: Complaint{Reason: ReasonCheating}, At: at}
}

func (s *StorageContractSuite) save(snapshot Snapshot) error {
	report, err := snapshot.Restore()
	s.Require().NoError(err)
	return s.storage.Save(s.T().Context(), report) //nolint:wrapcheck // the suite reads the port's own error.
}

func (s *StorageContractSuite) saved(name string, reporter AccountID, reported AccountID, at time.Time) Snapshot {
	snapshot := s.report(name, reporter, reported, at)
	s.Require().NoError(s.save(snapshot))
	return snapshot
}

func (s *StorageContractSuite) snapshots(found []Report, err error) []Snapshot {
	s.Require().NoError(err)

	taken := make([]Snapshot, 0, len(found))
	for _, report := range found {
		taken = append(taken, report.TakeSnapshot())
	}
	return taken
}

func (s *StorageContractSuite) same(want []Snapshot) {
	got := s.Kept(s.storage)

	s.Require().Len(got, len(want))
	for i := range want {
		s.True(want[i].At.Equal(got[i].At), "reported at %s, read %s", want[i].At, got[i].At)
		got[i].At = want[i].At
		s.Equal(want[i], got[i])

		_, err := got[i].Restore()
		s.NoError(err, "what a storage keeps restores a report")
	}
}

func (s *StorageContractSuite) TestNothingIsKeptAtFirst() {
	s.Empty(s.Kept(s.storage))
}

func (s *StorageContractSuite) TestASavedReportIsKeptAsItWas() {
	about := Snapshot{
		ID: contractID("message"), Reporter: contractAda, Reported: contractBob, At: contractStart,
		Complaint: Complaint{Reason: ReasonChatAbuse, Text: "it insults\neveryone 🤬", Message: "message-1"},
	}
	bare := Snapshot{
		ID: contractID("bare"), Reporter: contractEve, Reported: contractBob, At: contractStart.Add(time.Minute),
		Complaint: Complaint{Reason: ReasonBadName},
	}

	s.Require().NoError(s.save(about))
	s.Require().NoError(s.save(bare))

	s.same([]Snapshot{about, bare})
}

func (s *StorageContractSuite) TestAReportSavedAgainIsKeptOnce() {
	first := s.saved("first", contractAda, contractBob, contractStart)

	s.Require().NoError(s.save(first))

	s.same([]Snapshot{first})
}

func (s *StorageContractSuite) TestAStorageKeepsWhateverItIsHandedTheSameDayIncluded() {
	first := s.saved("first", contractAda, contractBob, contractStart)
	again := s.saved("again", contractAda, contractBob, contractStart.Add(time.Minute))

	s.same([]Snapshot{first, again})
}

func (s *StorageContractSuite) TestSeveralReportsAreSavedAtOnce() {
	first := s.report("first", contractAda, contractBob, contractStart)
	second := s.report("second", contractEve, contractBob, contractStart.Add(time.Minute))
	one, err := first.Restore()
	s.Require().NoError(err)
	two, err := second.Restore()
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Save(s.T().Context(), one, two))

	s.same([]Snapshot{first, second})
}

func (s *StorageContractSuite) TestSavingNothingIsNotAnError() {
	s.Require().NoError(s.storage.Save(s.T().Context()))
}

func (s *StorageContractSuite) TestAForgottenReportIsKeptNoMoreAndTheOthersStay() {
	gone := s.saved("gone", contractAda, contractBob, contractStart)
	kept := s.saved("kept", contractEve, contractBob, contractStart.Add(time.Minute))
	report, err := gone.Restore()
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Save(s.T().Context(), report.Forgotten()))

	s.same([]Snapshot{kept})
}

func (s *StorageContractSuite) TestForgettingAReportThatWasNeverKeptIsNotAnError() {
	report, err := s.report("never kept", contractAda, contractBob, contractStart).Restore()
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Save(s.T().Context(), report.Forgotten()))

	s.Empty(s.Kept(s.storage))
}

func (s *StorageContractSuite) TestWhatAReporterFiledAboutAPlayerComesOldestFirstAndNothingElse() {
	second := s.saved("second", contractAda, contractBob, contractStart.Add(24*time.Hour))
	s.saved("another reporter", contractEve, contractBob, contractStart.Add(time.Minute))
	s.saved("another player", contractAda, contractEve, contractStart.Add(2*time.Minute))
	s.saved("the other way round", contractBob, contractAda, contractStart.Add(3*time.Minute))
	first := s.saved("first", contractAda, contractBob, contractStart)

	s.Equal([]Snapshot{first, second}, s.snapshots(s.storage.FiledBy(s.T().Context(), contractAda, contractBob)))
	s.Empty(s.snapshots(s.storage.FiledBy(s.T().Context(), contractEve, contractAda)))
}

func (s *StorageContractSuite) TestTheReportsThatNameAnAccountAreTheOnesItFiledAndTheOnesAboutIt() {
	about := s.saved("about", contractEve, contractAda, contractStart.Add(time.Minute))
	s.saved("neither", contractEve, contractBob, contractStart.Add(2*time.Minute))
	filed := s.saved("filed", contractAda, contractBob, contractStart)

	s.Equal([]Snapshot{filed, about}, s.snapshots(s.storage.Naming(s.T().Context(), contractAda)))
	s.Empty(s.snapshots(s.storage.Naming(s.T().Context(), AccountID{15: 9})))
}

func (s *StorageContractSuite) TestTheReportsFiledBeforeATimeComeOldestFirstUpToTheLimit() {
	old := s.saved("old", contractAda, contractBob, contractStart)
	older := s.saved("older", contractEve, contractBob, contractStart.Add(-time.Hour))
	s.saved("at the cutoff", contractAda, contractEve, contractStart.Add(time.Hour))
	s.saved("later", contractBob, contractAda, contractStart.Add(2*time.Hour))

	s.Equal([]Snapshot{older, old}, s.snapshots(s.storage.FiledBefore(s.T().Context(), contractStart.Add(time.Hour), 10)))
	s.Equal([]Snapshot{older}, s.snapshots(s.storage.FiledBefore(s.T().Context(), contractStart.Add(time.Hour), 1)))
}
