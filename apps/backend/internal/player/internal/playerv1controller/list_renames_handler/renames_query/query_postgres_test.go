package renames_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/list_renames_handler/renames_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/renames/postgres_rename_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now      = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	ada      = players.AccountID{15: 1}
	bob      = players.AccountID{15: 2}
	operator = players.AccountID{15: 9}
	claude   = renames.OperatorOf(operator, "Claude")
)

type testSuite struct {
	suite.Suite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "player", migrations.FS)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) renamed(account players.AccountID, from, to players.Name, at time.Time, by renames.Operator) {
	before := players.ProfileSnapshot{Account: account, Name: from}.Restore()
	after := players.ProfileSnapshot{Account: account, Name: to}.Restore()
	s.Require().NoError(postgres_rename_store.New(s.db).Record(s.T().Context(),
		renames.NewRename(before, after, at, "a Nazi reference", by)))
}

func (s *testSuite) renames(filter renames_query.Filter) []*playerv1.Rename {
	answer, err := renames_query.NewPostgresQuery(s.db).Renames(s.T().Context(), filter)
	s.Require().NoError(err)
	for _, rename := range answer.GetRenames() {
		rename.RenameId = ""
	}
	return answer.GetRenames()
}

func (s *testSuite) TestEachRenameIsReadAsItWasMadeNewestFirst() {
	s.renamed(ada, "Hitler88", "Player42", now.Add(-time.Hour), claude)
	s.renamed(bob, "Slur", "Player7", now, renames.OperatorOf(players.AccountID{}, ""))

	got := s.renames(renames_query.Filter{})

	want := []*playerv1.Rename{
		{
			AccountId: bob.String(), RenamedAt: timestamppb.New(now), OldName: "Slur", NewName: "Player7",
			Reason: "a Nazi reference",
		},
		{
			AccountId: ada.String(), RenamedAt: timestamppb.New(now.Add(-time.Hour)), OldName: "Hitler88",
			NewName: "Player42", Reason: "a Nazi reference", OperatorAccountId: operator.String(), Client: "Claude",
		},
	}
	s.Require().Len(got, len(want))
	for i := range want {
		s.True(proto.Equal(want[i], got[i]), "rename %d is %v", i, got[i])
	}
}

func (s *testSuite) TestOneAccountsRenamesSinceATimeAreItsOwnAlone() {
	s.renamed(ada, "Hitler88", "Player42", now.Add(-8*24*time.Hour), claude)
	s.renamed(ada, "Player42", "Player43", now.Add(-7*24*time.Hour), claude)
	s.renamed(bob, "Slur", "Player7", now, claude)

	got := s.renames(renames_query.Filter{Account: ada, Since: now.Add(-7 * 24 * time.Hour)})

	s.Require().Len(got, 1)
	s.Equal("Player43", got[0].GetNewName())
}

func (s *testSuite) TestTheLimitKeepsTheNewest() {
	for i := range 3 {
		s.renamed(ada, "Hitler88", "Player42", now.Add(time.Duration(i)*time.Minute), claude)
	}

	got := s.renames(renames_query.Filter{Limit: 2})

	s.Require().Len(got, 2)
	s.Equal(now.Add(2*time.Minute), got[0].GetRenamedAt().AsTime())
}

func (s *testSuite) TestAPruneAndAForgetAreWhatTheReadSeesNoMore() {
	store := postgres_rename_store.New(s.db)
	s.renamed(ada, "Hitler88", "Player42", now.Add(-100*24*time.Hour), claude)
	s.renamed(ada, "Slur", "Player43", now, claude)
	s.renamed(bob, "Slur", "Player7", now, renames.OperatorOf(ada, ""))

	deleted, err := store.DeleteBefore(s.T().Context(), now.Add(-90*24*time.Hour))
	s.Require().NoError(err)
	s.Equal(int64(1), deleted)

	s.Require().NoError(store.Forget(s.T().Context(), ada))

	got := s.renames(renames_query.Filter{})
	s.Require().Len(got, 1, "the account forgotten loses its renames")
	s.Equal(bob.String(), got[0].GetAccountId())
	s.Empty(got[0].GetOperatorAccountId(), "and is taken off the ones it made")
}
