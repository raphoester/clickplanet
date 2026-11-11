//go:build testing

package cprules

import (
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

// StoreContractSuite is what every Store keeps, run with a module's own values.
type StoreContractSuite[M proto.Message] struct {
	suite.Suite

	NewStore func() Store[M]
	Parts    Parts[M]
	// Default is the module's default, kept before each test.
	Default Snapshot[M]
	// Rush is what a set "rush" holds: a scalar value of default's or a few, changed.
	Rush func() M
	// Other changes another scalar value, which Rush leaves to default.
	Other func() M

	store Store[M]
}

func (s *StoreContractSuite[M]) SetupTest() {
	s.store = s.NewStore()
	s.save(s.Default)
}

func (s *StoreContractSuite[M]) save(kept Snapshot[M]) {
	set, err := kept.Restore(s.Parts, s.Default)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), set))
}

func (s *StoreContractSuite[M]) set(name SetName) Set[M] {
	set, err := s.store.Set(s.T().Context(), name)
	s.Require().NoError(err)
	return set
}

// over is base with each change's values written over it.
func over[M proto.Message](base M, changes ...M) M {
	merged := proto.CloneOf(base)
	for _, change := range changes {
		proto.Merge(merged, change)
	}
	return merged
}

func (s *StoreContractSuite[M]) TestASetIsReadAsItWasKept() {
	rush := Snapshot[M]{Name: "rush", Rules: s.Rush()}
	s.save(rush)

	kept := s.set("rush").TakeSnapshot()
	s.Equal(SetName("rush"), kept.Name)
	s.True(proto.Equal(rush.Rules, kept.Rules), "%v", kept.Rules)
}

func (s *StoreContractSuite[M]) TestASetPlaysDefaultsValuesForThoseItLeavesUnset() {
	s.save(Snapshot[M]{Name: "rush", Rules: s.Rush()})

	played := s.set("rush").TakePlayedSnapshot().Rules
	s.True(proto.Equal(over(s.Default.Rules, s.Rush()), played), "%v", played)
}

func (s *StoreContractSuite[M]) TestASetFollowsDefaultWhereItHoldsNothing() {
	s.save(Snapshot[M]{Name: "rush", Rules: s.Rush()})
	s.save(Snapshot[M]{Name: Default, Rules: over(s.Default.Rules, s.Other())})

	played := s.set("rush").TakePlayedSnapshot().Rules
	s.True(proto.Equal(over(s.Default.Rules, s.Other(), s.Rush()), played), "%v", played)
}

func (s *StoreContractSuite[M]) TestASetNeverKeptIsUnknown() {
	_, err := s.store.Set(s.T().Context(), "frozen")
	s.Require().ErrorIs(err, ErrUnknownSet)
}

func (s *StoreContractSuite[M]) TestASetKeptAgainReplacesItsValues() {
	s.save(Snapshot[M]{Name: "rush", Rules: s.Rush()})
	s.save(Snapshot[M]{Name: "rush", Rules: s.Other()})

	kept := s.set("rush").TakeSnapshot().Rules
	s.True(proto.Equal(s.Other(), kept), "%v", kept)
}

func (s *StoreContractSuite[M]) TestNamesAreEverySetKept() {
	s.save(Snapshot[M]{Name: "rush", Rules: empty[M]()})

	names, err := s.store.Names(s.T().Context())
	s.Require().NoError(err)
	s.Equal([]SetName{Default, "rush"}, names)
}
