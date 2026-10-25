//go:build testing

package rules

import (
	"time"

	"github.com/stretchr/testify/suite"
)

type LayerStoreContractSuite struct {
	suite.Suite

	// NewStore is a store that holds the sets rush and default, which a layer names.
	NewStore func() LayerStore

	store LayerStore
}

func (s *LayerStoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractNoon = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

func (s *LayerStoreContractSuite) apply(request RequestID, set SetName, at time.Time) BaselineSnapshot {
	kept := BaselineSnapshot{Request: request, Set: set, AppliedAt: at}
	s.Require().NoError(s.store.SaveBaseline(s.T().Context(), kept.Restore()))
	return kept
}

func (s *LayerStoreContractSuite) TestNoBaselineIsNone() {
	latest, err := s.store.LatestBaseline(s.T().Context())
	s.Require().NoError(err)
	s.Equal(Baseline{}, latest)

	kept, err := s.store.Baseline(s.T().Context(), "season-0:finale")
	s.Require().NoError(err)
	s.Equal(Baseline{}, kept)
}

func (s *LayerStoreContractSuite) TestABaselineIsFoundByItsRequest() {
	finale := s.apply("season-0:finale", "rush", contractNoon)
	s.apply("season-0:over", "default", contractNoon.Add(time.Hour))

	kept, err := s.store.Baseline(s.T().Context(), "season-0:finale")
	s.Require().NoError(err)
	s.Equal(finale, kept.TakeSnapshot())
}

func (s *LayerStoreContractSuite) TestTheLatestBaselineIsTheOneAppliedLast() {
	s.apply("season-0:days", "default", contractNoon)
	over := s.apply("season-0:over", "default", contractNoon.Add(2*time.Hour))
	s.apply("season-0:finale", "rush", contractNoon.Add(time.Hour))

	latest, err := s.store.LatestBaseline(s.T().Context())
	s.Require().NoError(err)
	s.Equal(over, latest.TakeSnapshot())
}

func (s *LayerStoreContractSuite) TestOfTwoBaselinesAppliedAtOnceTheLatestIsTheOneSavedLast() {
	s.apply("operator:a", "rush", contractNoon)
	second := s.apply("operator:b", "default", contractNoon)

	latest, err := s.store.LatestBaseline(s.T().Context())
	s.Require().NoError(err)
	s.Equal(second, latest.TakeSnapshot())
}

func (s *LayerStoreContractSuite) TestARequestSavedAgainKeepsItsFirstBaseline() {
	first := s.apply("season-0:finale", "rush", contractNoon)
	s.apply("season-0:finale", "default", contractNoon.Add(time.Hour))

	kept, err := s.store.Baseline(s.T().Context(), "season-0:finale")
	s.Require().NoError(err)
	s.Equal(first, kept.TakeSnapshot())
}

func (s *LayerStoreContractSuite) TestNoOverrideIsNone() {
	kept, err := s.store.Override(s.T().Context())
	s.Require().NoError(err)
	s.Equal(Override{}, kept)
}

func (s *LayerStoreContractSuite) TestAnOverrideSavedAgainReplacesTheOneBefore() {
	first := OverrideSnapshot{Set: "rush", StartsAt: contractNoon, EndsAt: contractNoon.Add(time.Hour)}
	s.Require().NoError(s.store.SaveOverride(s.T().Context(), first.Restore()))
	ended := first
	ended.EndsAt = contractNoon.Add(time.Minute)
	s.Require().NoError(s.store.SaveOverride(s.T().Context(), ended.Restore()))

	kept, err := s.store.Override(s.T().Context())
	s.Require().NoError(err)
	s.Equal(ended, kept.TakeSnapshot())
}
