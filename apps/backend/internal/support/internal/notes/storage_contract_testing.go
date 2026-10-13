//go:build testing

package notes

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage
	// Every note the storage keeps, oldest first.
	Kept func(storage Storage) []Snapshot

	storage Storage
}

var contractStart = time.Date(2026, 10, 11, 12, 0, 0, 123_456_000, time.UTC)

var (
	contractAda = AccountID{15: 1}
	contractBob = AccountID{15: 2}
)

func contractID(name string) NoteID {
	return NoteID(uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)))
}

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) suggestion(name string, account AccountID, at time.Time) Snapshot {
	return Snapshot{ID: contractID(name), Kind: KindSuggestion, Account: account, Text: "a suggestion called " + name, At: at}
}

func (s *StorageContractSuite) save(snapshot Snapshot) error {
	note, err := snapshot.Restore()
	s.Require().NoError(err)
	return s.storage.Save(s.T().Context(), note) //nolint:wrapcheck // the suite reads the port's own error.
}

func (s *StorageContractSuite) saved(name string, account AccountID, at time.Time) Snapshot {
	snapshot := s.suggestion(name, account, at)
	s.Require().NoError(s.save(snapshot))
	return snapshot
}

func (s *StorageContractSuite) same(want []Snapshot) {
	got := s.Kept(s.storage)

	s.Require().Len(got, len(want))
	for i := range want {
		s.True(want[i].At.Equal(got[i].At), "submitted at %s, read %s", want[i].At, got[i].At)
		got[i].At = want[i].At
		s.Equal(want[i], got[i])

		_, err := got[i].Restore()
		s.NoError(err, "what a storage keeps restores a note")
	}
}

func (s *StorageContractSuite) TestNothingIsKeptAtFirst() {
	s.Empty(s.Kept(s.storage))
}

func (s *StorageContractSuite) TestASavedNoteIsKeptAsItWas() {
	note := Snapshot{
		ID: contractID("lines"), Kind: KindSuggestion, Account: contractAda,
		Text: "a night mode 🌙\nand a day mode", At: contractStart,
	}

	s.Require().NoError(s.save(note))

	s.same([]Snapshot{note})
}

var contractPhone = Diagnostics{
	Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, PixelRatio: 2.63, ConnectionDown: true,
	UserAgent: "Mozilla/5.0 (iPhone)", GPUVendor: "Apple Inc.", GPURenderer: "Apple GPU",
	WebGLVersion: "WebGL 2.0 (OpenGL ES 3.0)", MaxTextureSize: 4096, ContextLost: true, Cores: 6, MemoryGB: 0.5,
}

func (s *StorageContractSuite) TestANoteOfEitherKindIsKeptWithItsDiagnosticsAndFoundWithThem() {
	bug := Snapshot{
		ID: contractID("bug"), Kind: KindBug, Account: contractAda, Text: "the globe is black", At: contractStart,
		Diagnostics: contractPhone,
	}
	idea := Snapshot{
		ID: contractID("idea"), Kind: KindSuggestion, Account: contractAda, Text: "a night mode", At: contractStart.Add(time.Minute),
		Diagnostics: contractPhone,
	}
	bare := Snapshot{
		ID: contractID("bare"), Kind: KindBug, Account: contractAda, Text: "nothing happens", At: contractStart.Add(2 * time.Minute),
	}

	s.Require().NoError(s.save(bug))
	s.Require().NoError(s.save(idea))
	s.Require().NoError(s.save(bare))

	s.same([]Snapshot{bug, idea, bare})
	s.Equal([]Snapshot{bug, idea, bare}, s.snapshots(s.storage.SentBy(s.T().Context(), contractAda)))
}

func (s *StorageContractSuite) snapshots(found []Note, err error) []Snapshot {
	s.Require().NoError(err)

	taken := make([]Snapshot, 0, len(found))
	for _, note := range found {
		taken = append(taken, note.TakeSnapshot())
	}
	return taken
}

func (s *StorageContractSuite) TestANoteSavedAgainIsKeptOnce() {
	first := s.saved("first", contractAda, contractStart)

	s.Require().NoError(s.save(first))

	s.same([]Snapshot{first})
}

func (s *StorageContractSuite) TestSeveralNotesAreSavedAtOnce() {
	first := s.suggestion("first", contractAda, contractStart)
	second := s.suggestion("second", contractBob, contractStart.Add(time.Minute))
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

func (s *StorageContractSuite) anonymized(snapshot Snapshot) Snapshot {
	note, err := snapshot.Restore()
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Save(s.T().Context(), note.Anonymized()))

	snapshot.Account = AccountID{}
	snapshot.Attachments = nil
	return snapshot
}

func (s *StorageContractSuite) TestAnAnonymizedNoteIsKeptWithItsTextAndNoSender() {
	sent := s.saved("sent", contractAda, contractStart)
	kept := s.saved("kept", contractAda, contractStart.Add(time.Minute))

	anonymized := s.anonymized(sent)

	s.same([]Snapshot{anonymized, kept})
	s.Equal("a suggestion called sent", s.Kept(s.storage)[0].Text)
}

func (s *StorageContractSuite) TestAnAnonymizedNoteKeepsItsDiagnostics() {
	phone := Snapshot{
		ID: contractID("phone"), Kind: KindBug, Account: contractAda, Text: "the globe is black", At: contractStart,
		Diagnostics: contractPhone,
	}
	s.Require().NoError(s.save(phone))

	anonymized := s.anonymized(phone)

	s.same([]Snapshot{anonymized})
	s.Equal(contractPhone, s.Kept(s.storage)[0].Diagnostics)
}

func (s *StorageContractSuite) TestANoteSavedWithNoSenderFromTheStartIsKeptToo() {
	anonymized := s.anonymized(s.suggestion("never kept before", contractAda, contractStart))

	s.same([]Snapshot{anonymized})
}

func (s *StorageContractSuite) TestAnAnonymizedNoteIsSentByNobody() {
	s.anonymized(s.saved("sent", contractAda, contractStart))
	kept := s.saved("kept", contractAda, contractStart.Add(time.Minute))

	s.Equal([]Snapshot{kept}, s.snapshots(s.storage.SentBy(s.T().Context(), contractAda)))
	s.Empty(s.snapshots(s.storage.SentBy(s.T().Context(), AccountID{})))
}

func (s *StorageContractSuite) TestTheNotesAnAccountSentComeOldestFirstAndNobodyElses() {
	second := s.saved("second", contractAda, contractStart.Add(2*time.Minute))
	s.saved("another's", contractBob, contractStart.Add(time.Minute))
	first := s.saved("first", contractAda, contractStart)

	s.Equal([]Snapshot{first, second}, s.snapshots(s.storage.SentBy(s.T().Context(), contractAda)))
	s.Empty(s.snapshots(s.storage.SentBy(s.T().Context(), AccountID{15: 9})))
}

func (s *StorageContractSuite) TestTheNotesWithASenderBeforeATimeComeOldestFirstUpToTheLimit() {
	old := s.saved("old", contractAda, contractStart)
	older := s.saved("older", contractBob, contractStart.Add(-time.Hour))
	s.saved("at the cutoff", contractAda, contractStart.Add(time.Hour))
	s.saved("later", contractBob, contractStart.Add(2*time.Hour))

	s.Equal([]Snapshot{older, old}, s.snapshots(s.storage.WithSenderBefore(s.T().Context(), contractStart.Add(time.Hour), 10)))
	s.Equal([]Snapshot{older}, s.snapshots(s.storage.WithSenderBefore(s.T().Context(), contractStart.Add(time.Hour), 1)))
}

func (s *StorageContractSuite) TestANoteWithNoSenderIsNotAmongTheNotesWithOne() {
	s.anonymized(s.saved("oldest", contractAda, contractStart.Add(-2*time.Hour)))
	old := s.saved("old", contractBob, contractStart)

	s.Equal([]Snapshot{old}, s.snapshots(s.storage.WithSenderBefore(s.T().Context(), contractStart.Add(time.Hour), 10)))
}
