//go:build testing

package mutes

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type StorageContractSuite struct {
	suite.Suite

	NewStorage func() Storage

	storage Storage
}

var contractStart = time.Date(2024, 1, 1, 12, 0, 0, 123_456_000, time.UTC)

var (
	contractBully    = messages.AccountID{15: 1}
	contractNewGuest = messages.AccountID{15: 2}
	contractHome     = Scope("2001:db8:1:2::/64")
	contractNeighbor = Scope("2001:db8:1:3::/64")
)

func contractID(name string) MuteID {
	return MuteID(uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)))
}

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) save(name string, caller Caller, duration time.Duration) Mute {
	mute := NewMute(contractID(name), caller, contractStart, duration)
	s.Require().NoError(s.storage.Save(context.Background(), mute))
	return mute
}

func (s *StorageContractSuite) read(caller Caller, at time.Time) (Mute, error) {
	return s.storage.Mute(context.Background(), caller, at) //nolint:wrapcheck // the suite reads the port's own error.
}

func (s *StorageContractSuite) same(want Mute, got Mute) {
	s.Equal(want.ID(), got.ID())
	s.Equal(want.Caller(), got.Caller())
	s.True(want.At().Equal(got.At()), "muted at %s, read %s", want.At(), got.At())
	s.True(want.Until().Equal(got.Until()), "muted until %s, read %s", want.Until(), got.Until())
}

func (s *StorageContractSuite) TestNobodyIsMutedAtFirst() {
	_, err := s.read(CallerOf(contractBully, contractHome), contractStart)

	s.ErrorIs(err, ErrNotMuted)
}

func (s *StorageContractSuite) TestAMuteFollowsItsAccountToAnyNetwork() {
	mute := s.save("bully", CallerOf(contractBully, contractHome), time.Hour)

	got, err := s.read(CallerOf(contractBully, contractNeighbor), contractStart.Add(time.Minute))
	s.Require().NoError(err)

	s.same(mute, got)
}

func (s *StorageContractSuite) TestAMuteHoldsEveryAccountOnItsNetwork() {
	mute := s.save("bully", CallerOf(contractBully, contractHome), time.Hour)

	got, err := s.read(CallerOf(contractNewGuest, contractHome), contractStart.Add(time.Minute))
	s.Require().NoError(err)

	s.same(mute, got)
	_, err = s.read(CallerOf(contractNewGuest, contractNeighbor), contractStart.Add(time.Minute))
	s.ErrorIs(err, ErrNotMuted, "the next network is not muted")
}

func (s *StorageContractSuite) TestAMuteWithNoNetworkHoldsItsAccountAlone() {
	s.save("bully", CallerOf(contractBully, NoScope), time.Hour)

	_, err := s.read(CallerOf(contractNewGuest, NoScope), contractStart.Add(time.Minute))

	s.ErrorIs(err, ErrNotMuted, "no network is not one network")
}

func (s *StorageContractSuite) TestAMuteEndsAtItsUntil() {
	s.save("bully", CallerOf(contractBully, contractHome), time.Hour)

	_, err := s.read(CallerOf(contractBully, contractHome), contractStart.Add(time.Hour-time.Microsecond))
	s.Require().NoError(err)

	_, err = s.read(CallerOf(contractBully, contractHome), contractStart.Add(time.Hour))
	s.ErrorIs(err, ErrNotMuted)
}

func (s *StorageContractSuite) TestTheMuteThatEndsLastIsRead() {
	s.save("short", CallerOf(contractBully, NoScope), time.Hour)
	long := s.save("long", CallerOf(contractNewGuest, contractHome), 24*time.Hour)

	got, err := s.read(CallerOf(contractBully, contractHome), contractStart.Add(time.Minute))
	s.Require().NoError(err)

	s.same(long, got)
}

func (s *StorageContractSuite) TestAnIDIsKeptOnce() {
	s.save("bully", CallerOf(contractBully, contractHome), time.Hour)

	s.Error(s.storage.Save(context.Background(), NewMute(contractID("bully"), CallerOf(contractNewGuest, NoScope), contractStart, time.Hour)))
}

func (s *StorageContractSuite) TestDeleteBeforeRemovesTheMutesThatEndedAndCountsThem() {
	s.save("hour", CallerOf(contractBully, NoScope), time.Hour)
	s.save("day", CallerOf(contractNewGuest, NoScope), 24*time.Hour)

	deleted, err := s.storage.DeleteBefore(context.Background(), contractStart.Add(2*time.Hour))
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	_, err = s.read(CallerOf(contractNewGuest, NoScope), contractStart.Add(3*time.Hour))
	s.Require().NoError(err, "what ends later was kept")
}
