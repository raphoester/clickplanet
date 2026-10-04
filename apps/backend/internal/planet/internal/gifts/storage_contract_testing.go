//go:build testing

package gifts

import (
	"context"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage

	storage Storage
}

const (
	contractAda   bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"
	contractGrace bonuses.Holder = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60"
)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) TestAGiftIsGivenOnce() {
	s.Require().NoError(s.storage.Give(context.Background(), "finale-0", contractAda))

	s.Require().ErrorIs(s.storage.Give(context.Background(), "finale-0", contractAda), ErrGiven)
}

func (s *StorageContractSuite) TestEachAccountGetsItsOwn() {
	s.Require().NoError(s.storage.Give(context.Background(), "finale-0", contractAda))

	s.Require().NoError(s.storage.Give(context.Background(), "finale-0", contractGrace))
}

func (s *StorageContractSuite) TestAnotherTagIsAnotherGift() {
	s.Require().NoError(s.storage.Give(context.Background(), "finale-0", contractAda))

	s.Require().NoError(s.storage.Give(context.Background(), tempo.GiftTag("finale-1"), contractAda))
}
