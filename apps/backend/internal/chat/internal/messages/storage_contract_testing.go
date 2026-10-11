//go:build testing

package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/suite"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage

	storage Storage
}

var contractStart = time.Date(2024, 1, 1, 12, 0, 0, 123_456_000, time.UTC)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

var contractAccount = AccountID{15: 1}

func (s *StorageContractSuite) message(snapshot Snapshot) Message {
	message, err := snapshot.Restore()
	s.Require().NoError(err)
	return message
}

func (s *StorageContractSuite) keep(message Message, ip string) {
	s.Require().NoError(s.storage.Append(context.Background(), NewRecord(message, "some-uuid", ip, "test-agent")))
}

func (s *StorageContractSuite) sent(text string, at time.Time) Message {
	return s.message(Snapshot{ID: MessageID("id-" + text), SentAt: at, Account: contractAccount, Country: "fr", Text: text})
}

func (s *StorageContractSuite) append(texts ...string) {
	for i, text := range texts {
		s.keep(s.sent(text, contractStart.Add(time.Duration(i)*time.Hour)), "203.0.113.7")
	}
}

func (s *StorageContractSuite) shown(text string, since time.Time, limit int) bool {
	shown, err := s.storage.Shown(context.Background(), MessageID("id-"+text), since, limit)
	s.Require().NoError(err)
	return shown
}

func (s *StorageContractSuite) TestAnEmptyStorageShowsNothing() {
	s.False(s.shown("hello", contractStart, 10))
}

func (s *StorageContractSuite) TestAMessageAppendedIsShown() {
	s.append("hello")

	s.True(s.shown("hello", contractStart, 10))
}

func (s *StorageContractSuite) TestTheNewestIsTheLastAppendedWhateverItsTime() {
	s.keep(s.sent("first", contractStart.Add(time.Second)), "203.0.113.7")
	s.keep(s.sent("second", contractStart), "203.0.113.7")

	s.True(s.shown("second", contractStart, 1))
	s.False(s.shown("first", contractStart, 1))
}

func (s *StorageContractSuite) TestAMessageIsShownOnlyWhileItIsAmongTheNewestInTheWindow() {
	s.append("old", "middle", "new")

	s.True(s.shown("new", contractStart, 2))
	s.True(s.shown("middle", contractStart, 2))
	s.False(s.shown("old", contractStart, 2), "pushed out by newer ones")
	s.False(s.shown("middle", contractStart.Add(2*time.Hour), 10), "sent before the window")
	s.False(s.shown("never-sent", contractStart, 10))
}

func (s *StorageContractSuite) kept(id MessageID) Snapshot {
	message, err := s.storage.Message(context.Background(), id, contractStart, 10)
	s.Require().NoError(err)

	kept := message.TakeSnapshot()
	s.True(kept.SentAt.Equal(contractStart), "%s was sent at %s", id, kept.SentAt)
	kept.SentAt = contractStart
	return kept
}

func (s *StorageContractSuite) TestAShownMessageIsReadBackAsItWasKept() {
	s.append("hello")

	s.Equal(Snapshot{ID: "id-hello", SentAt: contractStart, Account: contractAccount, Country: "fr", Text: "hello"},
		s.kept("id-hello"))
}

func (s *StorageContractSuite) TestAMessageFromBeforeAccountsIsReadBackUnderTheNameItCarries() {
	legacy := Snapshot{
		ID: "id-legacy", SentAt: contractStart, Author: AuthorOf("guest_Bob", true, 0, 0, Title{}), Country: "fr", Text: "legacy",
	}
	s.keep(s.message(legacy), "203.0.113.7")

	s.Equal(legacy, s.kept("id-legacy"))
}

func (s *StorageContractSuite) TestAReplyIsReadBackWithTheIDOfWhatItAnswersAndNothingOfWhatThatSaid() {
	question := s.sent("question", contractStart).Named(AuthorOf("Ada", false, 0, 0, Title{}))
	s.keep(s.sent("answer", contractStart).Answering(question), "203.0.113.7")

	s.Equal(Quote{MessageID: "id-question"}, s.kept("id-answer").Quote)
}

func (s *StorageContractSuite) TestAMessageIsReadOnlyWhileItIsShown() {
	s.append("old", "middle", "new")

	for text, window := range map[string]struct {
		since time.Time
		limit int
	}{
		"old":        {contractStart, 2},
		"middle":     {contractStart.Add(2 * time.Hour), 10},
		"never-sent": {contractStart, 10},
	} {
		_, err := s.storage.Message(context.Background(), MessageID("id-"+text), window.since, window.limit)

		s.Require().ErrorIs(err, ErrNotShown, text)
		s.False(s.shown(text, window.since, window.limit), "%s: a message nobody may react to is not answered either", text)
	}
}

func (s *StorageContractSuite) TestTheLatestAddressIsTheOneTheAccountLastPostedFrom() {
	other := AccountID{15: 2}
	for i, sent := range []struct {
		account AccountID
		ip      string
	}{
		{contractAccount, "2001:db8:1:2::1"},
		{contractAccount, "2001:db8:9:9::1"},
		{other, "198.51.100.1"},
	} {
		s.keep(s.message(Snapshot{
			ID: MessageID(fmt.Sprintf("id-%d", i)), SentAt: contractStart.Add(-time.Duration(i) * time.Minute),
			Account: sent.account, Country: "fr", Text: "hi",
		}), sent.ip)
	}

	ip, err := s.storage.LatestAddress(context.Background(), contractAccount)
	s.Require().NoError(err)

	s.Equal("2001:db8:9:9::1", ip, "the last appended, whatever its time")
}

func (s *StorageContractSuite) TestAnAccountThatSentNothingHasNoLatestAddress() {
	s.append("hello")

	_, err := s.storage.LatestAddress(context.Background(), AccountID{15: 9})

	s.ErrorIs(err, ErrNoMessage)
}

func (s *StorageContractSuite) TestDeleteBeforeRemovesOnlyOlderMessages() {
	s.keep(s.sent("ancient", contractStart), "203.0.113.7")
	s.keep(s.sent("recent", contractStart.Add(48*time.Hour)), "203.0.113.7")

	deleted, err := s.storage.DeleteBefore(context.Background(), contractStart.Add(24*time.Hour))
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	s.True(s.shown("recent", contractStart, 10))
	s.False(s.shown("ancient", contractStart, 10))
}
