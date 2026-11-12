package postgres_rating_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings/postgres_rating_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var at = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

type kept struct {
	question string
	account  *string
	correct  bool
	timedOut bool
	ms       int64
	at       time.Time
}

type StoreSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_rating_store.Store
}

func TestTheStore(t *testing.T) {
	suite.Run(t, new(StoreSuite))
}

func (s *StoreSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "bonuses", migrations.FS)
	s.store = postgres_rating_store.New(s.db)
}

func (s *StoreSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func account(t *testing.T, id string) ratings.AccountID {
	t.Helper()

	account, err := ratings.AccountIDOf(id)
	require.NoError(t, err)
	return account
}

func (s *StoreSuite) answers() []kept {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(), `
		SELECT question_id, account_id::text, correct, timed_out, answer_ms, answered_at FROM quiz_answers ORDER BY answer_id`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var all []kept
	for rows.Next() {
		var each kept
		s.Require().NoError(rows.Scan(&each.question, &each.account, &each.correct, &each.timedOut, &each.ms, &each.at))
		each.at = each.at.UTC()
		all = append(all, each)
	}
	s.Require().NoError(rows.Err())
	return all
}

func (s *StoreSuite) TestEveryAnswerIsARowOfItsOwn() {
	ctx := s.T().Context()
	player := "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

	s.Require().NoError(s.store.SaveAnswer(ctx, ratings.Answer{
		Question: "capital:ee", Account: account(s.T(), player), Correct: true, After: 2500 * time.Millisecond, At: at,
	}))
	s.Require().NoError(s.store.SaveAnswer(ctx, ratings.Answer{
		Question: "capital:ee", Account: ratings.NoAccount, TimedOut: true, After: 8100 * time.Millisecond, At: at.Add(time.Minute),
	}))

	s.Equal([]kept{
		{question: "capital:ee", account: &player, correct: true, ms: 2500, at: at},
		{question: "capital:ee", account: nil, timedOut: true, ms: 8100, at: at.Add(time.Minute)},
	}, s.answers(), "a caller with no account answers with none")
}

func (s *StoreSuite) TestARatingIsWrittenOverTheOneBefore() {
	ctx := s.T().Context()

	s.Require().NoError(s.store.SaveRating(ctx, "capital:ee", ratings.RatingOf(0.5, 1, 1)))
	s.Require().NoError(s.store.SaveRating(ctx, "capital:ee", ratings.RatingOf(0.75, 2, 1)))
	s.Require().NoError(s.store.SaveRating(ctx, "capital:lv", ratings.RatingOf(-1.25, 1, 0)))

	kept, err := s.store.Ratings(ctx)
	s.Require().NoError(err)
	s.Equal(map[ratings.QuestionID]ratings.Rating{
		"capital:ee": ratings.RatingOf(0.75, 2, 1),
		"capital:lv": ratings.RatingOf(-1.25, 1, 0),
	}, kept)
}

func (s *StoreSuite) TestNoQuestionAnsweredIsNoRating() {
	kept, err := s.store.Ratings(s.T().Context())

	s.Require().NoError(err)
	s.Empty(kept)
}

func (s *StoreSuite) TestARatingPastAnyChanceIsRefused() {
	s.Require().Error(s.store.SaveRating(s.T().Context(), "capital:ee", ratings.RatingOf(50, 1, 1)))
	s.Require().Error(s.store.SaveRating(s.T().Context(), "capital:ee", ratings.RatingOf(0, 1, 2)), "more right than answers")
}

func (s *StoreSuite) TestARightTimeoutIsRefused() {
	s.Require().Error(s.store.SaveAnswer(s.T().Context(), ratings.Answer{Question: "capital:ee", Correct: true, TimedOut: true, At: at}))
}

func (s *StoreSuite) TestAForgottenAccountIsBlankedFromItsAnswersAlone() {
	ctx := s.T().Context()
	gone, stays := "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	for _, who := range []string{gone, stays, gone} {
		s.Require().NoError(s.store.SaveAnswer(ctx, ratings.Answer{Question: "capital:ee", Account: account(s.T(), who), At: at}))
	}

	s.Require().NoError(s.store.ForgetAccount(ctx, account(s.T(), gone)))

	answers := s.answers()
	s.Require().Len(answers, 3, "the answers stay: they still count for their question")
	s.Nil(answers[0].account)
	s.Equal(&stays, answers[1].account)
	s.Nil(answers[2].account)
}
