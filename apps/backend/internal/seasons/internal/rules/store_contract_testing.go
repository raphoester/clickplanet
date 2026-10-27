//go:build testing

package rules

import (
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) save(kept Snapshot) {
	set, err := kept.Restore()
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), set))
}

func (s *StoreContractSuite) kept(name SetName) Snapshot {
	set, err := s.store.Set(s.T().Context(), name)
	s.Require().NoError(err)
	return set.TakeSnapshot()
}

func (s *StoreContractSuite) TestASetIsReadAsItWasKept() {
	rush := Battle("rush", 3)
	s.save(rush)

	kept := s.kept("rush")
	s.Equal(SetName("rush"), kept.Name)
	s.True(proto.Equal(rush.Rules, kept.Rules), "%v", kept.Rules)
}

func (s *StoreContractSuite) TestASetNeverKeptIsUnknown() {
	_, err := s.store.Set(s.T().Context(), "frozen")
	s.Require().ErrorIs(err, ErrUnknownSet)
}

func (s *StoreContractSuite) TestASetKeptAgainReplacesItsValues() {
	s.save(Battle("rush", 3))
	s.save(Battle("rush", 5))

	s.True(proto.Equal(Battle("rush", 5).Rules, s.kept("rush").Rules))
}
