//go:build testing

package rallies

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type StoreContractSuite struct {
	suite.Suite

	NewStore func() Store

	store Store
}

func (s *StoreContractSuite) SetupTest() {
	s.store = s.NewStore()
}

var contractAt = time.Date(2026, 11, 16, 12, 30, 0, 123456000, time.UTC)

func contractAccount(n byte) cpsession.AccountID {
	return cpsession.AccountID(uuid.UUID{0x01, 0x92, 0x6c, 0x6e, 0x7a, 0x4b, 0x7c, 0x3d, 0x8e, 0x9f, 0, 0, 0, 0, 0, n})
}

func contractRally(n byte, flag string, endsAt time.Time) Snapshot {
	return Snapshot{
		ID: RallyID(uuid.UUID{0x01, 0x93, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, n}), Account: contractAccount(n), Flag: flag,
		Centre: clicks.Vec3{X: 0.6, Y: 0.8, Z: float64(n) / 1000}, Radius: 0.072, CalledAt: endsAt.Add(-3 * time.Minute), EndsAt: endsAt,
	}
}

func (s *StoreContractSuite) keep(kept Snapshot) {
	s.Require().NoError(s.store.Change(s.T().Context(), kept.Account, func(Rally) (Rally, error) {
		return kept.Restore(), nil
	}))
}

func (s *StoreContractSuite) last(account cpsession.AccountID) Snapshot {
	last, err := s.store.Last(s.T().Context(), account)
	s.Require().NoError(err)
	return last.TakeSnapshot()
}

func (s *StoreContractSuite) same(want Snapshot, got Snapshot) {
	s.True(want.CalledAt.Equal(got.CalledAt), "called at %v, not %v", got.CalledAt, want.CalledAt)
	s.True(want.EndsAt.Equal(got.EndsAt), "ends at %v, not %v", got.EndsAt, want.EndsAt)
	got.CalledAt, got.EndsAt, want.CalledAt, want.EndsAt = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	s.Equal(want, got)
}

func (s *StoreContractSuite) TestAnAccountThatNeverCalledHasNoRally() {
	s.Equal(Snapshot{}, s.last(contractAccount(1)))
}

func (s *StoreContractSuite) TestAChangeSeesTheLastAndReplacesIt() {
	first, next := contractRally(1, "fr", contractAt), contractRally(1, "de", contractAt.Add(time.Hour))
	s.keep(first)

	var seen Snapshot
	s.Require().NoError(s.store.Change(s.T().Context(), first.Account, func(last Rally) (Rally, error) {
		seen = last.TakeSnapshot()
		return next.Restore(), nil
	}))

	s.same(first, seen)
	s.same(next, s.last(first.Account))
}

func (s *StoreContractSuite) TestARefusedChangeKeepsTheLast() {
	kept := contractRally(1, "fr", contractAt)
	s.keep(kept)
	refused := errors.New("refused")

	err := s.store.Change(s.T().Context(), kept.Account, func(Rally) (Rally, error) {
		return contractRally(1, "de", contractAt.Add(time.Hour)).Restore(), refused
	})

	s.Require().ErrorIs(err, refused)
	s.same(kept, s.last(kept.Account))
}

func (s *StoreContractSuite) TestOnIsTheFlagsRalliesThatEndAfterTheTime() {
	s.keep(contractRally(1, "fr", contractAt.Add(time.Minute)))
	s.keep(contractRally(2, "fr", contractAt))
	s.keep(contractRally(3, "de", contractAt.Add(time.Minute)))
	s.keep(contractRally(4, "fr", contractAt.Add(2*time.Minute)))

	on, err := s.store.On(s.T().Context(), "fr", contractAt)
	s.Require().NoError(err)

	accounts := make([]cpsession.AccountID, 0, len(on))
	for _, rally := range on {
		accounts = append(accounts, rally.TakeSnapshot().Account)
	}
	s.ElementsMatch([]cpsession.AccountID{contractAccount(1), contractAccount(4)}, accounts,
		"one that ends at the time is over, and another flag's is not the flag's")
}
