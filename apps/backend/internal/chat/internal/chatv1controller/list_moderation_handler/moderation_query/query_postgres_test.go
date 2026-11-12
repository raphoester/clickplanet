package moderation_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_moderation_handler/moderation_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/postgres_moderation_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now      = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	ada      = messages.AccountID{15: 1}
	bob      = messages.AccountID{15: 2}
	operator = messages.AccountID{15: 9}
	claude   = moderation.OperatorOf(operator, "Claude")
	page     = moderation.OperatorOf(operator, "")
	gif      = messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 124}
)

type testSuite struct {
	suite.Suite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) recorded(actions ...moderation.Action) {
	for _, action := range actions {
		s.Require().NoError(postgres_moderation_store.New(s.db).Record(s.T().Context(), action))
	}
}

func (s *testSuite) actions(filter moderation_query.Filter) []*chatv1.ModerationAction {
	answer, err := moderation_query.NewPostgresQuery(s.db).Actions(s.T().Context(), filter)
	s.Require().NoError(err)
	for _, action := range answer.GetActions() {
		action.ActionId = ""
	}
	return answer.GetActions()
}

// Field by field, as proto.Equal reads them: a message carries state testify would compare too.
func (s *testSuite) sameActions(want []*chatv1.ModerationAction, got []*chatv1.ModerationAction, why string) {
	s.Require().Len(got, len(want), why)
	for i := range want {
		s.True(proto.Equal(want[i], got[i]), "%s: action %d is %v", why, i, got[i])
	}
}

func (s *testSuite) muteOf(account messages.AccountID, kind mutes.Kind, at time.Time, duration time.Duration) mutes.Mute {
	return mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(account, "2a01:e0a:1:2::/64"), kind, at, duration)
}

func (s *testSuite) removed(id messages.MessageID, snapshot messages.Snapshot, at time.Time, reason moderation.Reason) moderation.Action {
	snapshot.ID = id
	return moderation.NewRemoval(messages.Restored(s.T(), snapshot), at, reason, claude)
}

func (s *testSuite) TestEveryKindOfActionIsReadAsItWasTakenNewestFirst() {
	s.recorded(
		s.removed("insult", messages.Snapshot{SentAt: now.Add(-4 * time.Hour), Account: bob, Text: "you all suck"},
			now.Add(-3*time.Hour), "an insult"),
		s.removed("gore", messages.Snapshot{SentAt: now.Add(-3 * time.Hour), Account: bob, Gif: gif},
			now.Add(-2*time.Hour), "a gore GIF"),
		moderation.NewMute(s.muteOf(bob, mutes.Gifs, now.Add(-2*time.Hour), 15*time.Minute), "2nd offence", claude),
		moderation.NewUnmute(bob, mutes.Gifs, now.Add(-time.Hour), "", page),
	)

	s.sameActions([]*chatv1.ModerationAction{
		{
			AccountId: bob.String(), ActedAt: timestamppb.New(now.Add(-time.Hour)),
			Action:            &chatv1.ModerationAction_Unmute{Unmute: &chatv1.UnmuteAction{Kind: chatv1.MuteKind_MUTE_KIND_GIFS}},
			OperatorAccountId: operator.String(),
		},
		{
			AccountId: bob.String(), ActedAt: timestamppb.New(now.Add(-2 * time.Hour)),
			Action: &chatv1.ModerationAction_Mute{Mute: &chatv1.MuteAction{
				Kind: chatv1.MuteKind_MUTE_KIND_GIFS, Duration: durationpb.New(15 * time.Minute),
			}},
			Reason: "2nd offence", OperatorAccountId: operator.String(), Client: "Claude",
		},
		{
			AccountId: bob.String(), ActedAt: timestamppb.New(now.Add(-2 * time.Hour)),
			Action: &chatv1.ModerationAction_Removal{Removal: &chatv1.RemovalAction{
				MessageId: "gore",
				Body:      &chatv1.RemovalAction_Gif{Gif: &chatv1.Gif{Url: gif.URL, Width: gif.Width, Height: gif.Height}},
			}},
			Reason: "a gore GIF", OperatorAccountId: operator.String(), Client: "Claude",
		},
		{
			AccountId: bob.String(), ActedAt: timestamppb.New(now.Add(-3 * time.Hour)),
			Action: &chatv1.ModerationAction_Removal{Removal: &chatv1.RemovalAction{
				MessageId: "insult", Body: &chatv1.RemovalAction_Text{Text: "you all suck"},
			}},
			Reason: "an insult", OperatorAccountId: operator.String(), Client: "Claude",
		},
	}, s.actions(moderation_query.Filter{}), "two at one time: the later recorded first")
}

func (s *testSuite) TestOneAccountsActionsAreItsOwnAlone() {
	s.recorded(
		moderation.NewUnmute(ada, mutes.Chat, now, "", page),
		moderation.NewUnmute(bob, mutes.Chat, now, "", page),
	)

	actions := s.actions(moderation_query.Filter{Account: bob})

	s.Require().Len(actions, 1)
	s.Equal(bob.String(), actions[0].GetAccountId())
}

func (s *testSuite) TestOnlyTheActionsSinceTheTimeAskedAreRead() {
	s.recorded(
		moderation.NewUnmute(bob, mutes.Chat, now.Add(-8*24*time.Hour), "", page),
		moderation.NewUnmute(bob, mutes.Chat, now.Add(-7*24*time.Hour), "", page),
	)

	actions := s.actions(moderation_query.Filter{Account: bob, Since: now.Add(-7 * 24 * time.Hour)})

	s.Require().Len(actions, 1, "the time asked itself is in")
	s.Equal(now.Add(-7*24*time.Hour), actions[0].GetActedAt().AsTime())
}

func (s *testSuite) TestTheLimitKeepsTheNewest() {
	for i := range 3 {
		s.recorded(moderation.NewUnmute(bob, mutes.Chat, now.Add(time.Duration(i)*time.Minute), "", page))
	}

	actions := s.actions(moderation_query.Filter{Limit: 2})

	s.Require().Len(actions, 2)
	s.Equal(now.Add(2*time.Minute), actions[0].GetActedAt().AsTime())
	s.Equal(now.Add(time.Minute), actions[1].GetActedAt().AsTime())
}

func (s *testSuite) TestNoActionIsAnEmptyList() {
	answer, err := moderation_query.NewPostgresQuery(s.db).Actions(s.T().Context(), moderation_query.Filter{})
	s.Require().NoError(err)

	s.True(proto.Equal(&chatv1.ListModerationResponse{}, answer))
}
