package history_query_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
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

type commandAuthors struct {
	query *fakeAuthors
}

func authorOf(author *playerv1.Author) messages.Author {
	worn := author.GetWornTitle()
	rank := worn.GetRank()
	return messages.AuthorOf(author.GetName(), author.GetAdmin(), int32(author.GetColor()), author.GetStreak(),
		messages.TitleOf(worn.GetId(), worn.GetName(),
			messages.RankOf(rank.GetTrackId(), rank.GetTrackName(), rank.GetNumber(), rank.GetCount())))
}

func (c commandAuthors) Author(ctx context.Context, account messages.AccountID) (messages.Author, error) {
	found, err := c.Authors(ctx, []messages.AccountID{account})
	if err != nil {
		return messages.Author{}, err
	}
	return found[account], nil
}

func (c commandAuthors) Authors(ctx context.Context, accounts []messages.AccountID) (map[messages.AccountID]messages.Author, error) {
	found, err := c.query.Authors(ctx, accounts)
	if err != nil {
		return nil, err
	}
	named := make(map[messages.AccountID]messages.Author, len(found))
	for account, author := range found {
		named[account] = authorOf(author)
	}
	return named, nil
}

func (s *testSuite) TestAMessageIsInTheHistoryExactlyWhenItCanBeReactedTo() {
	for name, sends := range map[string][]time.Duration{
		"fewer than the window holds":   {time.Hour},
		"more than the window holds":    {4 * time.Hour, 3 * time.Hour, 2 * time.Hour, time.Hour},
		"kept out of time order":        {time.Hour, 3 * time.Hour, 2 * time.Hour},
		"across the retention":          {25 * time.Hour, 23 * time.Hour, time.Hour},
		"one just inside the retention": {historyRetention - time.Second, historyRetention + time.Second},
	} {
		s.Run(name, func() {
			s.Require().NoError(s.db.Purge(s.T().Context()))
			ids := make([]string, 0, len(sends))
			for i, ago := range sends {
				id := fmt.Sprintf("m%d", i)
				s.sentAt(id, ada, now.Add(-ago))
				ids = append(ids, id)
			}

			shown := cpcolls.NewSet[string]()
			for _, message := range s.history(ada).GetMessages() {
				shown.Add(message.GetId())
			}
			for _, id := range ids {
				reactable, err := s.messages.Shown(s.T().Context(), messages.MessageID(id), window.Since(now), window.Size())
				s.Require().NoError(err)
				s.Equal(reactable, shown.Contains(id), "%s: the history and React must agree on what is shown", id)
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
		s.authors.named[account] = &playerv1.Author{AccountId: account.String(), Name: fmt.Sprintf("Player%02d", i)}
		s.reacted("crowd", clown, reactions.ReactorOf(account), true, now.Add(time.Duration(i-30)*time.Minute))
	}

	const size = 3
	query := history_query.NewPostgresQuery(s.db, s.authors, cptime.NewFixedClock(now), size, historyRetention)
	react := react_usecase.New(s.messages, s.reactions, silentFeed{}, commandAuthors{query: s.authors},
		cptime.NewFixedClock(now), messages.NewWindow(size, historyRetention))
	untouched := reactions.Reaction(chatv1.Reaction_REACTION_EARTH)

	for _, viewer := range []messages.AccountID{ada, bob, gone} {
		history, err := query.History(s.T().Context(), viewer)
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
	s.authors.named[ada] = &playerv1.Author{
		AccountId: ada.String(), Name: "Ada_L", Admin: true, Color: playerv1.NameColor_NAME_COLOR_TEAL, Streak: 12,
		WornTitle: &playerv1.Title{
			Id: "settler", Name: "Settler", Rank: &playerv1.Rank{TrackId: "conquest", TrackName: "Conquest", Number: 1, Count: 5},
		},
	}
	s.authors.named[bob] = &playerv1.Author{AccountId: bob.String(), Name: "guest_0b1c2d"}
	clock := cptime.NewFixedClock(now.Add(-time.Hour).Add(123_456 * time.Microsecond))
	send := send_message_usecase.New(s.messages, silentFeed{}, cpcountries.New(), commandAuthors{query: s.authors},
		messages.NewDrafts(&messages.SequentialIDs{}, clock, messages.NewLimits(0)))

	answers := map[string]*chatv1.ChatMessage{}
	for _, in := range []send_message_usecase.In{
		{Account: ada, AuthorID: "browser-1", CountryID: "fr", Text: "  hello\tplanet  ", UserAgent: "test"},
		{Account: bob, AuthorID: "browser-2", CountryID: "de", Text: "hi", UserAgent: "test"},
	} {
		sent, err := send.Execute(s.T().Context(), in)
		s.Require().NoError(err)
		answers[string(sent.ID())] = chatmessage.Encode(sent, nil, 0)
		clock.Advance(time.Second)
	}

	history := s.history(ada).GetMessages()
	s.Require().Len(history, len(answers))
	for _, message := range history {
		s.True(proto.Equal(answers[message.GetId()], message), "SendMessage answered %v, the history reads %v",
			answers[message.GetId()], message)
	}
}

func (s *testSuite) TestEveryKindTheChatAnnouncesIsReadBack() {
	announce := announce_usecase.New(s.announcements, silentFeed{}, &announcements.SequentialIDs{})
	kinds := announcements.Kinds()
	for i, kind := range kinds {
		s.Require().NoError(announce.Execute(s.T().Context(), announce_usecase.In{
			Kind: kind, At: now.Add(-time.Duration(len(kinds)-i) * time.Minute), Payload: json.RawMessage(`{"country":"fr"}`),
		}))
	}

	everyKind, err := history_query.NewPostgresQuery(s.db, s.authors, cptime.NewFixedClock(now), len(kinds), historyRetention).
		History(s.T().Context(), ada)
	s.Require().NoError(err)

	read := make([]announcements.Kind, 0, len(kinds))
	for _, announcement := range everyKind.GetAnnouncements() {
		read = append(read, announcements.Kind(announcement.GetKind()))
	}
	s.Equal(kinds, read, "a kind the chat writes and the history refuses would empty the history")
}
