package postgres_charge_store_test

import (
	"context"
	"testing"

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

const (
	alice bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	bob   bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

func (s *testSuite) load() map[bonuses.Holder]bonuses.Held {
	hands := map[bonuses.Holder]bonuses.Held{}
	s.Require().NoError(s.store.Load(context.Background(), func(holder bonuses.Holder, hand bonuses.Held) {
		hands[holder] = hand
	}))
	return hands
}

func (s *testSuite) TestAnEmptyStoreLoadsNothing() {
	s.Empty(s.load())
}

func (s *testSuite) TestSaveThenLoadEveryHand() {
	hands := map[bonuses.Holder]bonuses.Held{
		alice: {Refill: true, Bomb: true, SpreadClicks: 5},
		bob:   {Enclosures: 2},
	}

	s.Require().NoError(s.store.Save(context.Background(), hands))

	s.Equal(hands, s.load())
}

func (s *testSuite) TestASecondSaveReplacesTheHand() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, map[bonuses.Holder]bonuses.Held{alice: {Bomb: true, Enclosures: 1}}))

	s.Require().NoError(s.store.Save(ctx, map[bonuses.Holder]bonuses.Held{alice: {Enclosures: 1}}))

	s.Equal(map[bonuses.Holder]bonuses.Held{alice: {Enclosures: 1}}, s.load())
}

func (s *testSuite) TestAZeroHandDeletesTheRow() {
	ctx := context.Background()
	s.Require().NoError(s.store.Save(ctx, map[bonuses.Holder]bonuses.Held{alice: {Bomb: true}, bob: {Bomb: true}}))

	s.Require().NoError(s.store.Save(ctx, map[bonuses.Holder]bonuses.Held{alice: {}}))

	s.Equal(map[bonuses.Holder]bonuses.Held{bob: {Bomb: true}}, s.load())
}

func (s *testSuite) TestDeletingAHandNeverStoredIsNotAnError() {
	s.Require().NoError(s.store.Save(context.Background(), map[bonuses.Holder]bonuses.Held{alice: {}}))

	s.Empty(s.load())
}
