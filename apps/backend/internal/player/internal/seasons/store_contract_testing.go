//go:build testing

package seasons

import (
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) record(account byte) RecordSnapshot {
	record, err := s.store.Record(s.T().Context(), players.AccountID{15: account})
	s.Require().NoError(err)
	return record.TakeSnapshot()
}

func (s *StoreContractSuite) save(account byte, played ...Season) {
	s.Require().NoError(s.store.Save(s.T().Context(), RecordSnapshot{Account: players.AccountID{15: account}, Seasons: played}.Restore()))
}

func (s *StoreContractSuite) TestAnAccountThatPlayedNoSeasonHasAnEmptyRecord() {
	s.Equal(RecordSnapshot{Account: players.AccountID{15: 1}}, s.record(1))
}

func (s *StoreContractSuite) TestARecordSavedIsReadBack() {
	s.save(1, Season{Number: 0, Flag: "fr", Won: true}, Season{Number: 1, Flag: "de"})

	s.Equal(RecordSnapshot{Account: players.AccountID{15: 1}, Seasons: []Season{
		{Number: 0, Flag: "fr", Won: true},
		{Number: 1, Flag: "de"},
	}}, s.record(1))
}

func (s *StoreContractSuite) TestASaveReplacesTheSeasonsKept() {
	s.save(1, Season{Number: 0, Flag: "fr"}, Season{Number: 1, Flag: "de"})

	s.save(1, Season{Number: 0, Flag: "it", Won: true})

	s.Equal([]Season{{Number: 0, Flag: "it", Won: true}}, s.record(1).Seasons)
}

func (s *StoreContractSuite) TestAForgottenRecordLeavesNoSeasonAndTheOthersStay() {
	s.save(1, Season{Number: 0, Flag: "fr"})
	s.save(2, Season{Number: 0, Flag: "de"})

	s.save(1)

	s.Empty(s.record(1).Seasons)
	s.Equal([]Season{{Number: 0, Flag: "de"}}, s.record(2).Seasons)
}

func (s *StoreContractSuite) TestThePageOfRecordsHoldsTheAccountsThatPlayed() {
	s.save(1, Season{Number: 0, Flag: "fr", Won: true})
	s.save(2, Season{Number: 0, Flag: "de"}, Season{Number: 1, Flag: "de"})
	s.save(3, Season{Number: 0, Flag: "it"})

	records, err := s.store.Records(s.T().Context(), []players.AccountID{{15: 1}, {15: 2}, {15: 4}})

	s.Require().NoError(err)
	s.Len(records, 2, "an account that played nothing is left out")
	s.Equal([]Season{{Number: 0, Flag: "fr", Won: true}}, records[players.AccountID{15: 1}].TakeSnapshot().Seasons)
	s.Equal([]Season{{Number: 0, Flag: "de"}, {Number: 1, Flag: "de"}}, records[players.AccountID{15: 2}].TakeSnapshot().Seasons)
}

func (s *StoreContractSuite) TestAnEmptyPageReadsNothing() {
	records, err := s.store.Records(s.T().Context(), nil)

	s.Require().NoError(err)
	s.Empty(records)
}
