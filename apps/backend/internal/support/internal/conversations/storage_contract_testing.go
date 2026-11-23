//go:build testing

package conversations

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage
	// Keeps a note, which a conversation is about.
	KeepNote func(storage Storage, note notes.NoteID)

	storage Storage
}

var contractStart = time.Date(2026, 11, 23, 12, 0, 0, 123_456_000, time.UTC)

func contractNote(name string) notes.NoteID {
	return notes.NoteID(uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)))
}

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) kept(name string) notes.NoteID {
	note := contractNote(name)
	s.KeepNote(s.storage, note)
	return note
}

func (s *StorageContractSuite) save(snapshots ...Snapshot) error {
	saved := make([]Conversation, 0, len(snapshots))
	for _, snapshot := range snapshots {
		conversation, err := snapshot.Restore()
		s.Require().NoError(err)
		saved = append(saved, conversation)
	}
	return s.storage.Save(s.T().Context(), saved...) //nolint:wrapcheck // the suite reads the port's own error.
}

func (s *StorageContractSuite) found(asked ...notes.NoteID) []Snapshot {
	found, err := s.storage.Conversations(s.T().Context(), asked...)
	s.Require().NoError(err)

	snapshots := make([]Snapshot, 0, len(found))
	for _, conversation := range found {
		snapshots = append(snapshots, conversation.TakeSnapshot())
	}
	return snapshots
}

func (s *StorageContractSuite) whole(note notes.NoteID) Snapshot {
	return Snapshot{
		Note: note,
		Answers: []Answer{
			{Text: "Thanks! Which phone?", Mail: "a1@clickplanet.lol", At: contractStart},
			{Text: "Fixed 🎉\nReload the page.", Mail: "a2@clickplanet.lol", At: contractStart.Add(2 * time.Hour)},
		},
		Replies: []Reply{
			{Mail: "r1@mail.example.com", From: "ada@example.com", Text: "An iPhone 13", At: contractStart.Add(time.Hour)},
		},
		ClosedAt: contractStart.Add(3 * time.Hour),
	}
}

func (s *StorageContractSuite) TestANoteNobodyWroteAboutHasAnEmptyConversation() {
	note := s.kept("quiet")

	s.Equal([]Snapshot{{Note: note}}, s.found(note))
}

func (s *StorageContractSuite) TestAConversationIsKeptAsItWasAndFoundInTheOrderAsked() {
	first, second, quiet := s.kept("first"), s.kept("second"), s.kept("quiet")
	answered := Snapshot{Note: second, Answers: []Answer{{Text: "Thanks!", Mail: "a3@clickplanet.lol", At: contractStart}}}

	s.Require().NoError(s.save(s.whole(first), answered))

	s.Equal([]Snapshot{answered, {Note: quiet}, s.whole(first)}, s.found(second, quiet, first))
}

func (s *StorageContractSuite) TestAConversationSavedAgainKeepsEachMailOnceAndTheLastClosing() {
	note := s.kept("again")
	s.Require().NoError(s.save(s.whole(note)))
	more := s.whole(note)
	more.Replies = append(more.Replies, Reply{Mail: "r2@mail.example.com", From: "ada@example.com", Text: "Works now", At: contractStart.Add(4 * time.Hour)})
	more.ClosedAt = contractStart.Add(5 * time.Hour)

	s.Require().NoError(s.save(more))
	s.Require().NoError(s.save(more))

	s.Equal([]Snapshot{more}, s.found(note))
}

func (s *StorageContractSuite) TestAnAnonymizedConversationKeepsTheWordsAndNoSender() {
	note := s.kept("anonymized")
	conversation, err := s.whole(note).Restore()
	s.Require().NoError(err)

	s.Require().NoError(s.storage.Save(s.T().Context(), conversation.Anonymized()))

	want := s.whole(note)
	want.Replies[0].From = ""
	s.Equal([]Snapshot{want}, s.found(note))
}

func (s *StorageContractSuite) TestAConversationAboutANoteNotKeptIsRefusedAndNothingOfTheSaveIsKept() {
	kept := s.kept("kept")

	err := s.save(s.whole(kept), s.whole(contractNote("never sent")))

	s.Require().ErrorIs(err, notes.ErrUnknownNote)
	s.Equal([]Snapshot{{Note: kept}}, s.found(kept))
}

func (s *StorageContractSuite) TestSavingNothingOrAnEmptyConversationIsNotAnError() {
	s.Require().NoError(s.storage.Save(s.T().Context()))
	s.Require().NoError(s.save(Snapshot{Note: contractNote("never sent")}))
}
