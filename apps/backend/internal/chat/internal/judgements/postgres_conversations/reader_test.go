package postgres_conversations_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/postgres_conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/postgres_removal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now      = time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)
	since    = now.Add(-30 * 24 * time.Hour)
	raider   = messages.AccountID{15: 1}
	somebody = messages.AccountID{15: 2}
	gif      = messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 124}
)

type testSuite struct {
	suite.Suite

	db     *cppg.Postgres
	reader *postgres_conversations.Reader
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	s.reader = postgres_conversations.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) said(id string, account messages.AccountID, room messages.Room, at time.Time, text string, shown messages.Gif) {
	s.Require().NoError(postgres_message_store.New(s.db).Append(s.T().Context(), messages.NewRecord(
		messages.Restored(s.T(), messages.Snapshot{
			ID: messages.MessageID(id), SentAt: at, Account: account, Room: room, Country: "fr", Text: text, Gif: shown,
		}), "203.0.113.7", "agent")))
}

func ids(lines []judgements.Line) []messages.MessageID {
	found := make([]messages.MessageID, 0, len(lines))
	for _, line := range lines {
		found = append(found, line.ID)
	}
	return found
}

func (s *testSuite) TestTheMessageIsReadWithItsRoomAroundItAndThePlayersOwnLines() {
	for i := range 30 {
		account := somebody
		if i%5 == 0 {
			account = raider
		}
		s.said(fmt.Sprintf("m-%02d", i), account, messages.World, now.Add(time.Duration(i-30)*time.Minute), "hi", messages.Gif{})
	}
	s.said("fr-1", raider, messages.FlagRoom("fr"), now.Add(-time.Minute), "", gif)
	s.Require().NoError(postgres_removal_store.New(s.db).Save(s.T().Context(),
		removals.Snapshot{MessageID: "m-20", RemovedAt: now}))

	conversation, err := s.reader.Conversation(s.T().Context(), "m-20", raider, since)

	s.Require().NoError(err)
	s.Require().NotNil(conversation.Reported)
	s.Equal(messages.MessageID("m-20"), conversation.Reported.ID)
	s.True(conversation.Reported.Removed)
	s.Equal(raider, conversation.Reported.Account)
	s.Len(conversation.Before, postgres_conversations.Before)
	s.Equal(messages.MessageID("m-05"), conversation.Before[0].ID, "oldest first")
	s.Equal(messages.MessageID("m-19"), conversation.Before[len(conversation.Before)-1].ID)
	s.Equal([]messages.MessageID{"m-21", "m-22", "m-23", "m-24", "m-25", "m-26", "m-27", "m-28", "m-29"}, ids(conversation.After),
		"the room's alone: the flag's room is not the world's")
	s.Equal([]messages.MessageID{"m-00", "m-05", "m-10", "m-15", "m-20", "m-25", "fr-1"}, ids(conversation.Own),
		"every room, oldest first")
	s.Equal(gif, conversation.Own[len(conversation.Own)-1].Gif)
}

func (s *testSuite) TestAReportOfAPlayerReadsItsOwnLinesAlone() {
	s.said("m-1", raider, messages.World, now.Add(-time.Minute), "hi", messages.Gif{})
	s.said("m-2", somebody, messages.World, now, "hello", messages.Gif{})

	conversation, err := s.reader.Conversation(s.T().Context(), judgements.NoMessage, raider, since)

	s.Require().NoError(err)
	s.Nil(conversation.Reported)
	s.Empty(conversation.Before)
	s.Equal([]messages.MessageID{"m-1"}, ids(conversation.Own))
}

func (s *testSuite) TestAMessagePastTheRetentionIsNotKept() {
	s.said("m-old", raider, messages.World, since.Add(-time.Minute), "hi", messages.Gif{})

	_, err := s.reader.Conversation(s.T().Context(), "m-old", raider, since)

	s.Require().ErrorIs(err, messages.ErrNotKept)
}
