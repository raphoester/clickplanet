//go:build testing

package inmemory_ban_storage

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
)

var contractNoon = time.Date(2026, 9, 16, 12, 0, 0, 123_456_000, time.UTC)

// PersistenceContractSuite is the behaviour every Persistence shares. Embed it and set NewPersistence.
type PersistenceContractSuite struct {
	suite.Suite

	// NewPersistence hands back an empty one, per test.
	NewPersistence func() Persistence

	persistence Persistence
}

func (s *PersistenceContractSuite) SetupTest() {
	s.persistence = s.NewPersistence()
}

func contractBan(tag string, reason string) bans.Ban {
	return bans.Ban{AuthorTag: tag, BannedAt: contractNoon, Reason: reason}
}

func (s *PersistenceContractSuite) upsert(ban bans.Ban) {
	s.Require().NoError(s.persistence.Upsert(context.Background(), ban))
}

func (s *PersistenceContractSuite) all() []bans.Ban {
	all, err := s.persistence.All(context.Background())
	s.Require().NoError(err)
	return all
}

func (s *PersistenceContractSuite) TestAnEmptyOneHoldsNoBans() {
	s.Empty(s.all())
}

func (s *PersistenceContractSuite) TestABanComesBackAsItWentIn() {
	s.upsert(contractBan("a1b2c3", "spam"))

	s.Equal([]bans.Ban{contractBan("a1b2c3", "spam")}, s.all())
}

// All is unordered, so this is the only claim it makes about more than one.
func (s *PersistenceContractSuite) TestEveryBanIsThere() {
	s.upsert(contractBan("a1b2c3", "spam"))
	s.upsert(contractBan("d4e5f6", "slurs"))

	s.ElementsMatch([]bans.Ban{contractBan("a1b2c3", "spam"), contractBan("d4e5f6", "slurs")}, s.all())
}

// The storage upserts a member it has already banned, to rewrite the reason.
func (s *PersistenceContractSuite) TestBanningTheSameTagRewritesItRatherThanFailing() {
	s.upsert(contractBan("a1b2c3", "spam"))
	s.upsert(contractBan("a1b2c3", "still spam"))

	s.Equal([]bans.Ban{contractBan("a1b2c3", "still spam")}, s.all())
}

func (s *PersistenceContractSuite) TestTheTimeSurvivesTheRoundTrip() {
	elsewhere := contractNoon.In(time.FixedZone("UTC+7", 7*60*60))
	s.upsert(bans.Ban{AuthorTag: "a1b2c3", BannedAt: elsewhere, Reason: "spam"})

	s.Require().Len(s.all(), 1)
	s.Equal(contractNoon, s.all()[0].BannedAt, "a ban read back must compare to the one written")
}

func (s *PersistenceContractSuite) TestDeletingLiftsOneBanAndLeavesTheRest() {
	s.upsert(contractBan("a1b2c3", "spam"))
	s.upsert(contractBan("d4e5f6", "spam"))

	s.Require().NoError(s.persistence.Delete(context.Background(), "a1b2c3"))

	s.Equal([]bans.Ban{contractBan("d4e5f6", "spam")}, s.all())
}

// A second Unban of the same member reaches this, and must not fail the call.
func (s *PersistenceContractSuite) TestDeletingWhatIsNotThereIsNotAnError() {
	s.Require().NoError(s.persistence.Delete(context.Background(), "a1b2c3"))
}

// Load hands what All returned straight into the set it keeps.
func (s *PersistenceContractSuite) TestWhatAllReturnsIsTheCallersToKeep() {
	s.upsert(contractBan("a1b2c3", "spam"))

	read := s.all()
	read[0].Reason = "tampered"

	s.Equal("spam", s.all()[0].Reason)
}
