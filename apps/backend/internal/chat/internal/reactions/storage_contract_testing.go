//go:build testing

package reactions

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// StorageContractSuite is the behaviour every Storage shares. Embed it and set NewStorage.
type StorageContractSuite struct {
	suite.Suite

	// NewStorage builds an empty storage.
	NewStorage func() Storage

	storage Storage
}

var contractStart = time.Date(2024, 1, 1, 12, 0, 0, 123_456_000, time.UTC)

const (
	contractLaugh Reaction = 1
	contractClown Reaction = 2

	contractAda Reactor = "account:ada"
	contractBo  Reactor = "guest:91aa3d"
)

func (s *StorageContractSuite) SetupTest() {
	s.storage = s.NewStorage()
}

func (s *StorageContractSuite) save(message messages.MessageID, reaction Reaction, reactor Reactor, on bool, at time.Time) {
	s.Require().NoError(s.storage.Save(context.Background(), Change{
		MessageID: message, Reaction: reaction, Reactor: reactor, On: on, At: at,
	}))
}

func (s *StorageContractSuite) reactions(ids ...messages.MessageID) map[messages.MessageID]Reactions {
	given, err := s.storage.Reactions(context.Background(), ids)
	s.Require().NoError(err)
	return given
}

func (s *StorageContractSuite) TestReactionsReadBackInTheOrderEachFirstAppeared() {
	s.save("hello", contractClown, contractAda, true, contractStart)
	s.save("hello", contractLaugh, contractBo, true, contractStart.Add(time.Second))
	s.save("hello", contractClown, contractBo, true, contractStart.Add(2*time.Second))

	s.Equal([]Count{
		{Reaction: contractClown, Count: 2, Mine: true, Reactors: []Reactor{contractAda, contractBo}},
		{Reaction: contractLaugh, Count: 1, Reactors: []Reactor{contractBo}},
	}, s.reactions("hello")["hello"].Tally(contractAda))
}

func (s *StorageContractSuite) TestOnlyTheMessagesAskedForAreAnswered() {
	s.save("hello", contractClown, contractAda, true, contractStart)
	s.save("planet", contractClown, contractAda, true, contractStart)

	given := s.reactions("hello", "bare")

	s.Len(given, 1, "a message never reacted to is absent")
	s.True(given["hello"].Given(contractClown, contractAda))
	s.Empty(s.reactions())
}

func (s *StorageContractSuite) TestAReactionSavedTwiceIsOne() {
	s.save("hello", contractClown, contractAda, true, contractStart)
	s.save("hello", contractClown, contractAda, true, contractStart.Add(time.Hour))

	s.Equal([]Count{
		{Reaction: contractClown, Count: 1, Reactors: []Reactor{contractAda}},
	}, s.reactions("hello")["hello"].Tally(NoReactor))
}

func (s *StorageContractSuite) TestTakingOffRemovesOnlyThatOne() {
	s.save("hello", contractClown, contractAda, true, contractStart)
	s.save("hello", contractClown, contractBo, true, contractStart)

	s.save("hello", contractClown, contractAda, false, contractStart)
	s.save("hello", contractClown, contractAda, false, contractStart)

	given := s.reactions("hello")["hello"]
	s.False(given.Given(contractClown, contractAda))
	s.True(given.Given(contractClown, contractBo))
}

func (s *StorageContractSuite) TestDeleteBeforeRemovesOnlyOlderReactions() {
	s.save("hello", contractClown, contractAda, true, contractStart)
	s.save("hello", contractLaugh, contractAda, true, contractStart.Add(48*time.Hour))

	deleted, err := s.storage.DeleteBefore(context.Background(), contractStart.Add(24*time.Hour))
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	s.Equal([]Count{
		{Reaction: contractLaugh, Count: 1, Reactors: []Reactor{contractAda}},
	}, s.reactions("hello")["hello"].Tally(NoReactor))
	s.Equal(uint64(2), s.reactions("hello")["hello"].Version(), "a prune is not a change")
}

func (s *StorageContractSuite) TestEachChangeBumpsTheVersionAndANoOpDoesNot() {
	s.save("hello", contractClown, contractAda, true, contractStart)
	s.Equal(uint64(1), s.reactions("hello")["hello"].Version())

	s.save("hello", contractClown, contractAda, true, contractStart)
	s.save("hello", contractLaugh, contractAda, false, contractStart)
	s.Equal(uint64(1), s.reactions("hello")["hello"].Version(), "nothing changed")

	s.save("hello", contractLaugh, contractBo, true, contractStart)
	s.save("hello", contractClown, contractAda, false, contractStart)
	s.Equal(uint64(3), s.reactions("hello")["hello"].Version())
}

func (s *StorageContractSuite) TestAMessageWhoseReactionsWereAllTakenOffKeepsItsVersion() {
	s.save("hello", contractClown, contractAda, true, contractStart)
	s.save("hello", contractClown, contractAda, false, contractStart)

	given, answered := s.reactions("hello")["hello"]

	s.True(answered, "a client holding version 1 must learn of version 2")
	s.Empty(given.Tally(NoReactor))
	s.Equal(uint64(2), given.Version())
}

func (s *StorageContractSuite) TestEachMessageHasItsOwnVersion() {
	s.save("hello", contractClown, contractAda, true, contractStart)
	s.save("hello", contractLaugh, contractAda, true, contractStart)
	s.save("planet", contractClown, contractAda, true, contractStart)

	given := s.reactions("hello", "planet")

	s.Equal(uint64(2), given["hello"].Version())
	s.Equal(uint64(1), given["planet"].Version())
}

func (s *StorageContractSuite) TestATallySaysWhoGaveEachReactionOldestFirst() {
	s.save("hello", contractClown, contractBo, true, contractStart)
	s.save("hello", contractClown, contractAda, true, contractStart.Add(time.Second))

	counts := s.reactions("hello")["hello"].Tally(NoReactor)

	s.Equal([]Reactor{contractBo, contractAda}, counts[0].Reactors,
		"accounts, not names: the chat keeps no copy of one")
}
