package shown_names_query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_shown_names_handler/shown_names_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/postgres_optout_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/postgres_verdict_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/screen_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

type usernames struct {
	names map[cpsession.AccountID]string
	err   error
	asked [][]cpsession.AccountID
}

func (u *usernames) Of(_ context.Context, accounts []cpsession.AccountID) (map[cpsession.AccountID]string, error) {
	u.asked = append(u.asked, accounts)
	held := map[cpsession.AccountID]string{}
	for _, account := range accounts {
		if name, named := u.names[account]; named {
			held[account] = name
		}
	}
	return held, u.err
}

type ShownNamesSuite struct {
	suite.Suite

	db       *cppg.Postgres
	verdicts *postgres_verdict_store.Store
	optOuts  *postgres_optout_store.Store

	ada, ira, bob, cyd, eve, dan, guest cpsession.AccountID
	held                                *usernames
}

func TestShownNamesSuite(t *testing.T) {
	suite.Run(t, new(ShownNamesSuite))
}

func (s *ShownNamesSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "clips", migrations.FS)
	s.verdicts, s.optOuts = postgres_verdict_store.New(s.db), postgres_optout_store.New(s.db)
}

func (s *ShownNamesSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.ada, s.ira, s.bob, s.cyd = cpsession.AccountID(uuid.New()), cpsession.AccountID(uuid.New()),
		cpsession.AccountID(uuid.New()), cpsession.AccountID(uuid.New())
	s.eve, s.dan, s.guest = cpsession.AccountID(uuid.New()), cpsession.AccountID(uuid.New()), cpsession.AccountID(uuid.New())

	s.Require().NoError(s.verdicts.Keep(s.T().Context(), verdicts.Judged(map[verdicts.AccountID]string{
		s.ada: "Ada", s.ira: "IRA4Ever", s.bob: "Bob", s.cyd: "Cyd", s.dan: "Dan88", s.guest: "Gus",
	}, []verdicts.Judgement{
		{Name: "Ada", Shown: true},
		{Name: "IRA4Ever", Reason: "the IRA, an armed group"},
		{Name: "Bob", Shown: true},
		{Name: "Cyd", Shown: true},
		{Name: "Dan88", Reason: "a hate code"},
		{Name: "Gus", Shown: true},
	}, at)))
	hidden, err := s.verdicts.Verdict(s.T().Context(), s.dan)
	s.Require().NoError(err)
	s.Require().NoError(s.verdicts.Keep(s.T().Context(), []verdicts.Verdict{hidden.Overruled(true, "", at.Add(time.Hour))}))
	s.Require().NoError(s.optOuts.Keep(s.T().Context(), s.bob))

	s.held = &usernames{names: map[cpsession.AccountID]string{
		s.ada: "Ada", s.ira: "IRA4Ever", s.bob: "Bob", s.cyd: "Cyd_2", s.eve: "Eve", s.dan: "Dan88",
	}}
}

func (s *ShownNamesSuite) everyone() []cpsession.AccountID {
	return []cpsession.AccountID{s.dan, s.ira, s.bob, s.cyd, s.eve, s.guest, s.ada}
}

func (s *ShownNamesSuite) TestOnlyANameShownThatThePlayerHoldsNowAndKeepsInIsAnswered() {
	answer, err := shown_names_query.NewPostgresQuery(s.db, s.held).ShownNames(s.T().Context(), s.everyone())
	s.Require().NoError(err)

	s.True(proto.Equal(&clipsv1.GetShownNamesResponse{ShownNames: []*clipsv1.ShownName{
		{AccountId: s.dan.String(), Name: "Dan88"},
		{AccountId: s.ada.String(), Name: "Ada"},
	}}, answer), "in the order asked: %s", answer)
	s.Equal([][]cpsession.AccountID{s.everyone()}, s.held.asked)
}

func (s *ShownNamesSuite) TestARepeatedAccountIsAskedAndAnsweredOnce() {
	answer, err := shown_names_query.NewPostgresQuery(s.db, s.held).
		ShownNames(s.T().Context(), []cpsession.AccountID{s.ada, s.eve, s.ada})
	s.Require().NoError(err)

	s.True(proto.Equal(&clipsv1.GetShownNamesResponse{ShownNames: []*clipsv1.ShownName{
		{AccountId: s.ada.String(), Name: "Ada"},
	}}, answer), answer.String())
	s.Equal([][]cpsession.AccountID{{s.ada, s.eve}}, s.held.asked)
}

func (s *ShownNamesSuite) TestNoAccountAsksNobody() {
	answer, err := shown_names_query.NewPostgresQuery(s.db, s.held).ShownNames(s.T().Context(), []cpsession.AccountID{})
	s.Require().NoError(err)

	s.Empty(answer.GetShownNames())
	s.Empty(s.held.asked)
}

func (s *ShownNamesSuite) TestUsernamesThatCannotBeReadFailTheRead() {
	failure := errors.New("player is down")
	s.held.err = failure

	_, err := shown_names_query.NewPostgresQuery(s.db, s.held).ShownNames(s.T().Context(), s.everyone())

	s.Require().ErrorIs(err, failure)
}

func (s *ShownNamesSuite) TestTheReadShowsTheNamesACastShowsWhenTheJudgeAnswersNothing() {
	judge := verdicts.NewScriptedJudge()
	cast, err := casts.NewShownNames(s.held, s.optOuts, screen_names_usecase.New(s.verdicts, judge, cptime.NewFixedClock(at))).
		Shown(s.T().Context(), s.everyone())
	s.Require().NoError(err)

	answer, err := shown_names_query.NewPostgresQuery(s.db, s.held).ShownNames(s.T().Context(), s.everyone())
	s.Require().NoError(err)

	read := map[cpsession.AccountID]string{}
	for _, shown := range answer.GetShownNames() {
		account, err := casts.AccountIDOf(shown.GetAccountId())
		s.Require().NoError(err)
		read[account] = shown.GetName()
	}
	s.Equal(map[cpsession.AccountID]string{s.dan: "Dan88", s.ada: "Ada"}, read)
	s.Equal(cast, read)
	s.Equal([][]string{{"Cyd_2", "Eve"}}, judge.Asked, "the cast asked the judge of what has no verdict")
}
