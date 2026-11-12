//go:build testing

package arrivals

import (
	"github.com/stretchr/testify/suite"
)

type RecruitmentStoreContractSuite struct {
	suite.Suite

	NewStore func() Recruitments
	// Every recruitment the store keeps, by arrival.
	Kept func(store Recruitments) []RecruitmentSnapshot

	store Recruitments
}

func (s *RecruitmentStoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *RecruitmentStoreContractSuite) save(snapshots ...RecruitmentSnapshot) {
	saved := make([]Recruitment, 0, len(snapshots))
	for _, snapshot := range snapshots {
		recruitment, err := snapshot.Restore()
		s.Require().NoError(err)
		saved = append(saved, recruitment)
	}
	s.Require().NoError(s.store.Save(s.T().Context(), saved...))
}

func (s *RecruitmentStoreContractSuite) TestNothingIsKeptAtFirst() {
	s.Empty(s.Kept(s.store))
}

func (s *RecruitmentStoreContractSuite) TestARecruitmentIsKeptAsItWas() {
	first := RecruitmentSnapshot{Arrival: contractID("first"), Recruiter: contractAda}
	second := RecruitmentSnapshot{Arrival: contractID("second"), Recruiter: contractBob}

	s.save(first, second)

	s.ElementsMatch([]RecruitmentSnapshot{first, second}, s.Kept(s.store))
}

func (s *RecruitmentStoreContractSuite) TestARecruitmentSavedAgainReplacesItself() {
	recruitment := RecruitmentSnapshot{Arrival: contractID("once"), Recruiter: contractAda}
	s.save(recruitment)

	anonymized := recruitment
	anonymized.Recruiter = AccountID{}
	s.save(anonymized)

	s.Equal([]RecruitmentSnapshot{anonymized}, s.Kept(s.store))
}

func (s *RecruitmentStoreContractSuite) TestTheRecruitmentsOfARecruiterAreFound() {
	first := RecruitmentSnapshot{Arrival: contractID("first"), Recruiter: contractAda}
	second := RecruitmentSnapshot{Arrival: contractID("second"), Recruiter: contractAda}
	s.save(first, RecruitmentSnapshot{Arrival: contractID("other"), Recruiter: contractBob}, second)

	found, err := s.store.By(s.T().Context(), contractAda)

	s.Require().NoError(err)
	snapshots := make([]RecruitmentSnapshot, 0, len(found))
	for _, recruitment := range found {
		snapshots = append(snapshots, recruitment.TakeSnapshot())
	}
	s.ElementsMatch([]RecruitmentSnapshot{first, second}, snapshots)
}

func (s *RecruitmentStoreContractSuite) TestAnAnonymizedRecruitmentNamesNobody() {
	s.save(RecruitmentSnapshot{Arrival: contractID("gone")})

	found, err := s.store.By(s.T().Context(), AccountID{})

	s.Require().NoError(err)
	s.Empty(found)
}
