package outbox_bonuses_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/outbox_bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/postgres_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
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
		return outbox_bonuses.New(outbox).KeepUnlocks(ctx, ada, []levels.Kind{"enclose_clicks", "bomb"}, monday)
	}))

	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&bonusesv1.KeepUnlocks{
		AccountId: ada.String(),
		Kinds:     []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS, planetv1.BonusKind_BONUS_KIND_BOMB},
		DecidedAt: timestamppb.New(monday),
	}, outbox.Recorded()[0]))
}

func TestALevelReachedIsALevelUpInTheBonusesWordsNamedByTheAccountAndTheLevel(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}
	pay := func(account accounts.AccountID, level levels.Level) {
		require.NoError(t, cpatomic.FakeTransactioner{}.InTx(t.Context(), func(ctx context.Context) error {
			return outbox_bonuses.New(outbox).PayLevelUp(ctx, account, level, []levels.Kind{"enclose_clicks"})
		}))
	}

	pay(ada, 4)
	pay(ada, 4)
	pay(ada, 5)
	pay(accounts.AccountID{15: 2}, 4)

	recorded := outbox.Recorded()
	require.Len(t, recorded, 4)
	first, ok := recorded[0].(*bonusesv1.PayLevelUp)
	require.True(t, ok)
	assert.Equal(t, ada.String(), first.GetAccountId())
	assert.Equal(t, uint32(4), first.GetLevel())
	assert.Equal(t, []planetv1.BonusKind{planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS}, first.GetUnlocked())
	grants := cpcolls.NewSet[string]()
	for _, each := range recorded {
		levelUp, ok := each.(*bonusesv1.PayLevelUp)
		require.True(t, ok)
		grants.Add(levelUp.GetGrantId())
	}
	assert.Equal(t, 3, grants.Len(), "one grant per account and level: the same level twice is one grant")
}

func TestAKindPlanetDoesNotNameIsRefused(t *testing.T) {
	err := cpatomic.FakeTransactioner{}.InTx(t.Context(), func(ctx context.Context) error {
		return outbox_bonuses.New(&cpatomic.FakeOutbox{}).KeepUnlocks(ctx, ada, []levels.Kind{"bounty"}, monday)
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
	book := levels.NewBook(s.db, postgres_level_store.New(s.db), outbox_bonuses.New(cpoutbox.New(s.db)))

	_, err := book.Raise(s.T().Context(), ada, 5, monday)
	s.Require().NoError(err)

	kept, err := postgres_level_store.New(s.db).Levels(s.T().Context(), []accounts.AccountID{ada})
	s.Require().NoError(err)
	s.Equal(levels.ByAccount{ada: 5}, kept)
	s.Equal(2, s.outboxRows(), "its unlocks and its level-up")
}

// refusing records the commands, then fails: the raise must go with them.
type refusing struct {
	inner levels.Bonuses
}

func (r refusing) PayLevelUp(ctx context.Context, account accounts.AccountID, level levels.Level, unlocked []levels.Kind) error {
	if err := r.inner.PayLevelUp(ctx, account, level, unlocked); err != nil {
		return err //nolint:wrapcheck // the test reads it.
	}
	return errors.New("the command failed after it was recorded")
}

func (r refusing) KeepUnlocks(ctx context.Context, account accounts.AccountID, kinds []levels.Kind, at time.Time) error {
	if err := r.inner.KeepUnlocks(ctx, account, kinds, at); err != nil {
		return err //nolint:wrapcheck // the test reads it.
	}
	return errors.New("the command failed after it was recorded")
}

func (s *postgresSuite) TestARaiseWhoseCommandFailsIsNotKept() {
	book := levels.NewBook(s.db, postgres_level_store.New(s.db), refusing{inner: outbox_bonuses.New(cpoutbox.New(s.db))})

	_, err := book.Raise(s.T().Context(), ada, 5, monday)
	s.Require().Error(err)

	kept, err := postgres_level_store.New(s.db).Levels(s.T().Context(), []accounts.AccountID{ada})
	s.Require().NoError(err)
	s.Empty(kept)
	s.Zero(s.outboxRows())
}
