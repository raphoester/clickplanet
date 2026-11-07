package postgres_moderation_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation/postgres_moderation_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	at       = time.Date(2026, 11, 6, 12, 0, 0, 123_456_000, time.UTC)
	ada      = messages.AccountID{15: 1}
	bob      = messages.AccountID{15: 2}
	operator = messages.AccountID{15: 9}
	claude   = moderation.OperatorOf(operator, "Claude")
)

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_moderation_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	s.store = postgres_moderation_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

type row struct {
	account  uuid.NullUUID
	action   string
	operator uuid.NullUUID
	client   string
}

func (s *testSuite) kept() []row {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(),
		`SELECT account_id, action, operator_id, client FROM moderation ORDER BY id`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []row
	for rows.Next() {
		var each row
		s.Require().NoError(rows.Scan(&each.account, &each.action, &each.operator, &each.client))
		kept = append(kept, each)
	}
	s.Require().NoError(rows.Err())
	return kept
}

func (s *testSuite) removal(id messages.MessageID, account messages.AccountID, at time.Time) moderation.Action {
	message := messages.Restored(s.T(), messages.Snapshot{ID: id, SentAt: at, Account: account, Text: "you all suck"})
	return moderation.NewRemoval(message, at, "an insult", claude)
}

func (s *testSuite) TestAMessageIsRemovedOnceInTheHistory() {
	s.Require().NoError(s.store.Record(s.T().Context(), s.removal("insult", bob, at)))
	s.Require().NoError(s.store.Record(s.T().Context(), s.removal("insult", bob, at.Add(time.Minute))))

	s.Len(s.kept(), 1, "a call tried again never counts twice")
}

func (s *testSuite) TestEveryMuteIsAnActionOfItsOwn() {
	mute := mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(bob, mutes.NoScope), mutes.Chat, at, time.Hour)
	s.Require().NoError(s.store.Record(s.T().Context(), moderation.NewMute(mute, "", claude)))
	s.Require().NoError(s.store.Record(s.T().Context(), moderation.NewMute(mute, "", claude)))

	s.Len(s.kept(), 2)
}

func (s *testSuite) TestTheConsoleIsNobody() {
	s.Require().NoError(s.store.Record(s.T().Context(),
		moderation.NewUnmute(bob, mutes.Gifs, at, "", moderation.OperatorOf(messages.NoAccount, ""))))

	s.Equal([]row{{
		account: uuid.NullUUID{UUID: uuid.UUID(bob), Valid: true}, action: "unmute",
	}}, s.kept())
}

func (s *testSuite) TestAPruneDeletesOnlyWhatWasDoneBeforeTheCutoff() {
	s.Require().NoError(s.store.Record(s.T().Context(), s.removal("old", bob, at.Add(-time.Hour))))
	s.Require().NoError(s.store.Record(s.T().Context(), s.removal("new", bob, at)))

	deleted, err := s.store.DeleteBefore(s.T().Context(), at)
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	s.Len(s.kept(), 1)
}

func (s *testSuite) TestAForgottenAccountLosesItsHistoryAndIsTakenOffWhatItDidAsAnOperator() {
	s.Require().NoError(s.store.Record(s.T().Context(), s.removal("insult", ada, at)))
	s.Require().NoError(s.store.Record(s.T().Context(), s.removal("spam", bob, at)))
	s.Require().NoError(s.store.Record(s.T().Context(),
		moderation.NewUnmute(ada, mutes.Chat, at, "", moderation.OperatorOf(bob, ""))))

	s.Require().NoError(s.store.Forget(s.T().Context(), ada))
	s.Require().NoError(s.store.Forget(s.T().Context(), operator))

	s.Equal([]row{{
		account: uuid.NullUUID{UUID: uuid.UUID(bob), Valid: true}, action: "removal", client: "Claude",
	}}, s.kept(), "the action on Bob stays, by nobody now")
}

func (s *testSuite) TestTheActionsOnAnAccountAreReadBackSinceATimeNewestFirst() {
	judge := moderation.OperatorOf(messages.NoAccount, "Claude judge")
	mute := mutes.NewMute(mutes.MuteID{15: 1}, mutes.CallerOf(bob, mutes.NoScope), mutes.Gifs, at, 15*time.Minute)
	s.Require().NoError(s.store.Record(s.T().Context(), s.removal("old", bob, at.Add(-8*24*time.Hour))))
	s.Require().NoError(s.store.Record(s.T().Context(), moderation.NewRemoval(
		messages.Restored(s.T(), messages.Snapshot{ID: "gif", SentAt: at, Account: bob, Text: ""}), at, "offence 2 (gifs): gore", judge)))
	s.Require().NoError(s.store.Record(s.T().Context(), moderation.NewMute(mute, "offence 2 (gifs): gore", judge)))
	s.Require().NoError(s.store.Record(s.T().Context(), s.removal("ada's", ada, at)))

	actions, err := s.store.ActionsOn(s.T().Context(), bob, at.Add(-7*24*time.Hour))

	s.Require().NoError(err)
	s.Equal([]moderation.Snapshot{
		{
			Account: bob, At: at, Kind: moderation.Muted, MuteKind: mutes.Gifs, Duration: 15 * time.Minute,
			Reason: "offence 2 (gifs): gore", By: judge,
		},
		{Account: bob, At: at, Kind: moderation.Removed, Message: "gif", Reason: "offence 2 (gifs): gore", By: judge},
	}, actions, "never another account's, nor one past the window, nor a copy of what was said")
}
