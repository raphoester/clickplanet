//go:build testing

package grants

import (
	"time"

	"github.com/stretchr/testify/suite"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store
	// Every grant the store keeps, by id.
	Kept func(store Store) []Snapshot
	// An account a grant may name. A store whose grants belong to accounts it keeps elsewhere makes it.
	NewAccount func(account AccountID)

	store Store
}

var contractStart = time.Date(2026, 11, 3, 12, 0, 0, 123_456_000, time.UTC)

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

func (s *StoreContractSuite) grant(id byte, account byte, at time.Time) Grant {
	s.NewAccount(AccountID{15: account})
	grant, err := NewGrant(GrantID{15: id}, AccountID{15: account}, "4242", "the-access-token", at)
	s.Require().NoError(err)
	return grant
}

func (s *StoreContractSuite) same(want Snapshot, got Snapshot) {
	s.True(want.ExpiresAt.Equal(got.ExpiresAt), "expires at %s, read %s", want.ExpiresAt, got.ExpiresAt)
	got.ExpiresAt = want.ExpiresAt
	s.Equal(want, got)
}

func (s *StoreContractSuite) TestNothingIsKeptAtFirst() {
	s.Empty(s.Kept(s.store))
}

func (s *StoreContractSuite) TestAGrantIsReadBackAsItWasSaved() {
	grant := s.grant(1, 1, contractStart)
	s.Require().NoError(s.store.SaveGrant(s.T().Context(), grant))

	found, err := s.store.Grant(s.T().Context(), grant.ID())

	s.Require().NoError(err)
	s.same(grant.TakeSnapshot(), found.TakeSnapshot())
}

func (s *StoreContractSuite) TestAGrantNobodyKeptIsNotFound() {
	s.Require().NoError(s.store.SaveGrant(s.T().Context(), s.grant(1, 1, contractStart)))

	_, err := s.store.Grant(s.T().Context(), GrantID{15: 2})

	s.ErrorIs(err, ErrGrantNotFound)
}

func (s *StoreContractSuite) TestAPruneDeletesTheExpiredGrantsOnly() {
	expired := s.grant(1, 1, contractStart)
	lapsing := s.grant(2, 1, contractStart.Add(time.Minute))
	live := s.grant(3, 2, contractStart.Add(2*time.Minute))
	for _, grant := range []Grant{expired, lapsing, live} {
		s.Require().NoError(s.store.SaveGrant(s.T().Context(), grant))
	}

	pruned, err := s.store.PruneGrants(s.T().Context(), contractStart.Add(Lifetime+time.Minute), 10)

	s.Require().NoError(err)
	s.Equal(2, pruned, "a grant expired at the very instant is pruned")
	kept := s.Kept(s.store)
	s.Require().Len(kept, 1)
	s.same(live.TakeSnapshot(), kept[0])
	_, err = s.store.Grant(s.T().Context(), expired.ID())
	s.ErrorIs(err, ErrGrantNotFound)
}

func (s *StoreContractSuite) TestAPruneDeletesNoMoreThanItsLimit() {
	for id := range byte(3) {
		s.Require().NoError(s.store.SaveGrant(s.T().Context(), s.grant(id+1, 1, contractStart)))
	}

	pruned, err := s.store.PruneGrants(s.T().Context(), contractStart.Add(Lifetime), 2)

	s.Require().NoError(err)
	s.Equal(2, pruned)
	s.Len(s.Kept(s.store), 1)
}
