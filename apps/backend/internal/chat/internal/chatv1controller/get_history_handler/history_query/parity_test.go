package history_query_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler/history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type silentFeed struct{}

func (silentFeed) Publish(feed.Update) {}

type everyoneAPlayer struct{}

func (everyoneAPlayer) Author(context.Context, messages.AccountID) (messages.Author, error) {
	return messages.AuthorOf("Ada_L"), nil
}

func (s *testSuite) TestAMessageIsInTheHistoryExactlyWhenItCanBeReactedAndRepliedTo() {
	for name, kept := range map[string]struct {
		sends   []time.Duration
		removed []string
	}{
		"fewer than the window holds":   {sends: []time.Duration{time.Hour}},
		"more than the window holds":    {sends: []time.Duration{4 * time.Hour, 3 * time.Hour, 2 * time.Hour, time.Hour}},
		"kept out of time order":        {sends: []time.Duration{time.Hour, 3 * time.Hour, 2 * time.Hour}},
		"across the retention":          {sends: []time.Duration{25 * time.Hour, 23 * time.Hour, time.Hour}},
		"one just inside the retention": {sends: []time.Duration{historyRetention - time.Second, historyRetention + time.Second}},
		"one removed in the window": {
			sends: []time.Duration{4 * time.Hour, 3 * time.Hour, 2 * time.Hour, time.Hour}, removed: []string{"m3"},
		},
		"one removed past the window": {
			sends: []time.Duration{4 * time.Hour, 3 * time.Hour, 2 * time.Hour, time.Hour}, removed: []string{"m0"},
		},
	} {
		s.Run(name, func() {
			s.Require().NoError(s.db.Purge(s.T().Context()))
			ids := make([]string, 0, len(kept.sends))
			for i, ago := range kept.sends {
				id := fmt.Sprintf("m%d", i)
				s.sentAt(id, ada, now.Add(-ago))
				ids = append(ids, id)
			}
			for _, id := range kept.removed {
				s.removed(id)
			}

			shown := cpcolls.NewSet[string]()
			for _, message := range s.history(ada).GetMessages() {
				if !message.GetRemoved() {
					shown.Add(message.GetId())
				}
			}
			for _, id := range ids {
				_, err := s.messages.Message(s.T().Context(), messages.MessageID(id), window.Since(now), window.Size())
				s.Equal(!errors.Is(err, messages.ErrNotShown), shown.Contains(id),
					"%s: the history, React and SendMessage must agree on what is shown", id)
			}
		})
	}
}

func (s *testSuite) TestTheReactionsOfAMessageAreWhatReactAnswersForThem() {
	gone := messages.AccountID{15: 9}
	s.sent("hello", 3*time.Hour)
	s.sent("crowd", 2*time.Hour)
	s.sent("bare", time.Hour)
	s.reacted("hello", clown, reactions.ReactorOf(bob), true, now.Add(-90*time.Minute))
	s.reacted("hello", laugh, reactions.ReactorOf(ada), true, now.Add(-80*time.Minute))
	s.reacted("hello", clown, reactions.ReactorOf(ada), true, now.Add(-70*time.Minute))
	s.reacted("hello", clown, reactions.ReactorOf(gone), true, now.Add(-60*time.Minute))
	s.reacted("hello", clown, "guest:91aa3d", true, now.Add(-50*time.Minute))
	s.reacted("hello", laugh, reactions.ReactorOf(bob), true, now.Add(-40*time.Minute))
	s.reacted("hello", laugh, reactions.ReactorOf(bob), false, now.Add(-30*time.Minute))
	for i := range history_query.NamedReactors + 1 {
		account := messages.AccountID{14: 1, 15: byte(i)}
		s.reacted("crowd", clown, reactions.ReactorOf(account), true, now.Add(time.Duration(i-30)*time.Minute))
	}

	const size = 3
	query := history_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now), size, historyRetention)
	shown := messages.NewShowing(s.messages, messages.NewRooms(messages.FakeFlags{}), cptime.NewFixedClock(now),
		messages.NewWindow(size, historyRetention))
	react := react_usecase.New(shown, s.reactions, silentFeed{}, cptime.NewFixedClock(now))
	untouched := reactions.Reaction(chatv1.Reaction_REACTION_EARTH)

	for _, viewer := range []messages.AccountID{ada, bob, gone} {
		history, err := query.History(s.T().Context(), viewer, messages.World)
		s.Require().NoError(err)
		s.Require().Len(history.GetMessages(), size)
		for _, message := range history.GetMessages() {
			out, err := react.Execute(s.T().Context(), react_usecase.In{
				Account: viewer, MessageID: messages.MessageID(message.GetId()), Reaction: untouched, On: false,
			})
			s.Require().NoError(err)

			s.True(proto.Equal(
				&chatv1.ReactResponse{Reactions: chatmessage.EncodeCounts(out.Counts), Version: out.Version},
				&chatv1.ReactResponse{Reactions: message.GetReactions(), Version: message.GetReactionsVersion()},
			), "%s seen by %s: React answers %v, the history %v", message.GetId(), viewer, out.Counts, message.GetReactions())
		}
	}
}

func (s *testSuite) TestAMessageReadsBackAsSendMessageAnsweredIt() {
	clock := cptime.NewFixedClock(now.Add(-time.Hour).Add(123_456 * time.Microsecond))
	send := send_message_usecase.New(s.messages, silentFeed{}, everyoneAPlayer{},
		messages.NewDrafts(&messages.SequentialIDs{}, clock, messages.NewLimits(0), cpcountries.New(),
			messages.NewRooms(messages.FakeFlags{})),
		messages.NewThread(s.messages, clock, window))

	answers := map[string]*chatv1.ChatMessage{}
	for _, in := range []send_message_usecase.In{
		{Account: ada, Room: messages.World, CountryID: "fr", Text: "  hello\tplanet  ", UserAgent: "test"},
		{Account: bob, Room: messages.World, CountryID: "de", Text: "hi", UserAgent: "test"},
	} {
		sent, err := send.Execute(s.T().Context(), in)
		s.Require().NoError(err)
		answer := chatmessage.Encode(sent)
		answers[answer.GetId()] = answer
		clock.Advance(time.Second)
	}

	history := s.history(ada).GetMessages()
	s.Require().Len(history, len(answers))
	for _, message := range history {
		s.True(proto.Equal(answers[message.GetId()], message), "SendMessage answered %v, the history reads %v",
			answers[message.GetId()], message)
	}
}

func (s *testSuite) TestAReplyReadsBackAsSendMessageAnsweredIt() {
	const size = 10
	long := strings.Repeat("north ", 40)
	s.keep(messages.Snapshot{ID: "long", SentAt: now.Add(-5 * time.Hour), Account: bob, Country: "fr", Text: long})
	s.keep(messages.Snapshot{ID: "short", SentAt: now.Add(-4 * time.Hour), Account: bob, Country: "fr", Text: "who holds the north"})
	s.legacy(now.Add(-3 * time.Hour))

	clock := cptime.NewFixedClock(now.Add(-time.Hour))
	send := send_message_usecase.New(s.messages, silentFeed{}, everyoneAPlayer{},
		messages.NewDrafts(&messages.SequentialIDs{}, clock, messages.NewLimits(0), cpcountries.New(),
			messages.NewRooms(messages.FakeFlags{})),
		messages.NewThread(s.messages, clock, messages.NewWindow(size, historyRetention)))

	answers := map[string]*chatv1.ChatMessage{}
	for _, replyTo := range []messages.MessageID{"long", "short", "legacy"} {
		sent, err := send.Execute(s.T().Context(), send_message_usecase.In{
			Account: ada, Room: messages.World, CountryID: "fr", Text: "answer", ReplyTo: replyTo, UserAgent: "test",
		})
		s.Require().NoError(err)
		answer := chatmessage.Encode(sent)
		s.Require().Equal(string(replyTo), answer.GetReplyTo().GetMessageId())
		answers[answer.GetId()] = answer
		clock.Advance(time.Second)
	}

	history, err := history_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now), size, historyRetention).
		History(s.T().Context(), ada, messages.World)
	s.Require().NoError(err)
	replies := 0
	for _, message := range history.GetMessages() {
		answer, sent := answers[message.GetId()]
		if !sent {
			continue
		}
		replies++
		s.True(proto.Equal(answer, message), "SendMessage answered %v, the history reads %v", answer, message)
	}
	s.Equal(len(answers), replies)
}

func (s *testSuite) TestEveryKindTheChatAnnouncesIsReadBack() {
	announce := announce_usecase.New(s.announcements, silentFeed{})
	kinds := announcements.Kinds()
	read := make([]announcements.Kind, 0, len(kinds))
	for i, kind := range kinds {
		s.Require().NoError(announce.Execute(s.T().Context(), announcements.NewAnnouncement(
			announcements.AnnouncementID{15: byte(i + 1)}, kind, now.Add(-time.Duration(len(kinds)-i)*time.Minute),
			json.RawMessage(`{"country":"fr"}`))))
		shown := s.history(ada).GetAnnouncements()
		s.Require().NotEmpty(shown)
		read = append(read, announcements.Kind(shown[len(shown)-1].GetKind()))
	}

	s.Equal(kinds, read, "a kind the chat writes and the history refuses would empty the history")
}
