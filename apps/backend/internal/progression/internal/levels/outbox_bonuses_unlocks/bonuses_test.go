package outbox_bonuses_unlocks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/outbox_bonuses_unlocks"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/postgres_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

var (
	ada    = accounts.AccountID{15: 1}
	monday = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
)

func TestWhatALevelUnlocksIsACommandInTheBonusesWords(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}

	require.NoError(t, cpatomic.FakeTransactioner{}.InTx(t.Context(), func(ctx context.Context) error {
		return outbox_bonuses_unlocks.New(outbox).KeepUnlocks(ctx, ada, []levels.Kind{"enclose_clicks", "bomb"}, monday)
	}))

	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&bonusesv1.KeepUnlocks{
		AccountId: ada.String(),
		Kinds:     []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS, planetv1.BonusKind_BONUS_KIND_BOMB},
		DecidedAt: timestamppb.New(monday),
	}, outbox.Recorded()[0]))
}

func TestAKindPlanetDoesNotNameIsRefused(t *testing.T) {
	err := cpatomic.FakeTransactioner{}.InTx(t.Context(), func(ctx context.Context) error {
		return outbox_bonuses_unlocks.New(&cpatomic.FakeOutbox{}).KeepUnlocks(ctx, ada, []levels.Kind{"rally"}, monday)
	})

	require.Error(t, err)
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(postgresSuite))
}

type postgresSuite struct {
	suite.Suite

	db *cppg.Postgres
}

func (s *postgresSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "progression", migrations.FS)
}

func (s *postgresSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *postgresSuite) outboxRows() int {
	var rows int
	s.Require().NoError(s.db.On(s.T().Context()).QueryRowContext(s.T().Context(), `SELECT count(*) FROM outbox`).Scan(&rows))
	return rows
}

func (s *postgresSuite) TestARaiseAndItsCommandAreKeptTogether() {
	book := levels.NewBook(s.db, postgres_level_store.New(s.db), outbox_bonuses_unlocks.New(cpoutbox.New(s.db)))

	_, err := book.Raise(s.T().Context(), ada, 5, monday)
	s.Require().NoError(err)

	kept, err := postgres_level_store.New(s.db).Levels(s.T().Context(), []accounts.AccountID{ada})
	s.Require().NoError(err)
	s.Equal(levels.ByAccount{ada: 5}, kept)
	s.Equal(1, s.outboxRows())
}

// refusing records the command, then fails: the raise must go with it.
type refusing struct {
	inner levels.Bonuses
}

func (r refusing) KeepUnlocks(ctx context.Context, account accounts.AccountID, kinds []levels.Kind, at time.Time) error {
	if err := r.inner.KeepUnlocks(ctx, account, kinds, at); err != nil {
		return err //nolint:wrapcheck // the test reads it.
	}
	return errors.New("the command failed after it was recorded")
}

func (s *postgresSuite) TestARaiseWhoseCommandFailsIsNotKept() {
	book := levels.NewBook(s.db, postgres_level_store.New(s.db), refusing{inner: outbox_bonuses_unlocks.New(cpoutbox.New(s.db))})

	_, err := book.Raise(s.T().Context(), ada, 5, monday)
	s.Require().Error(err)

	kept, err := postgres_level_store.New(s.db).Levels(s.T().Context(), []accounts.AccountID{ada})
	s.Require().NoError(err)
	s.Empty(kept)
	s.Zero(s.outboxRows())
}
