package postgres_charge_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/postgres_charge_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	db    *cppg.Postgres
	store *postgres_charge_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "planet", migrations.FS)
	s.store = postgres_charge_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

var until = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

const (
	alice bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	bob   bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

func (s *testSuite) load() map[bonuses.Holder]bonuses.Hand {
	hands := map[bonuses.Holder]bonuses.Hand{}
	s.Require().NoError(s.store.Load(context.Background(), func(holder bonuses.Holder, hand bonuses.Hand) {
		hands[holder] = hand
	}))
	return hands
}

func (s *testSuite) TestAnEmptyStoreLoadsNothing() {
	s.Empty(s.load())
}

func (s *testSuite) TestSaveThenLoadEveryHand() {
	hands := map[bonuses.Holder]bonuses.Hand{
		alice: {Refill: until.Add(time.Minute), Bomb: until, Spread: until.Add(time.Hour), SpreadClicks: 5},
		bob:   {Enclose: until},
	}

	s.Require().NoError(s.store.Save(context.Background(), hands))

	s.Equal(hands, s.load(), "a kind not held comes back as a zero time")
}

func (s *testSuite) TestASecondSaveReplacesTheHand() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, map[bonuses.Holder]bonuses.Hand{alice: {Bomb: until, Enclose: until}}))

	s.Require().NoError(s.store.Save(ctx, map[bonuses.Holder]bonuses.Hand{alice: {Enclose: until}}))

	s.Equal(map[bonuses.Holder]bonuses.Hand{alice: {Enclose: until}}, s.load())
}

func (s *testSuite) TestAZeroHandDeletesTheRow() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, map[bonuses.Holder]bonuses.Hand{alice: {Bomb: until}, bob: {Bomb: until}}))

	s.Require().NoError(s.store.Save(ctx, map[bonuses.Holder]bonuses.Hand{alice: {}}))

	s.Equal(map[bonuses.Holder]bonuses.Hand{bob: {Bomb: until}}, s.load())
}

func (s *testSuite) TestDeletingAHandNeverStoredIsNotAnError() {
	s.Require().NoError(s.store.Save(context.Background(), map[bonuses.Holder]bonuses.Hand{alice: {}}))

	s.Empty(s.load())
}
