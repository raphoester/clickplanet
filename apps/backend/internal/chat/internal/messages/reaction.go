package messages

import (
	"errors"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Reaction is one of chat.v1.Reaction, by its wire number. The number is what is stored, which is why the proto
// never renumbers one. The edge refuses a number the proto does not name.
type Reaction int32

// Reactor is who put a reaction on. A player reacts as its account and a guest as the tag of its address, the
// one its messages carry. Each kind has its own prefix, so the two cannot be mistaken for each other.
type Reactor string

// NoReactor is nobody: what the stream tallies for, since it cannot know who reads it.
const NoReactor Reactor = ""

// ReactorOf is who reacts, from the same answer that names who posts: a player with a username is its account,
// anyone else the tag of its address. Two guests behind one address are therefore one reactor.
func ReactorOf(account AccountID, author Author) Reactor {
	if author.PostsAsPlayer(account) {
		return Reactor("account:" + account.String())
	}
	return Reactor("guest:" + author.Tag)
}

// PostsAsPlayer is a sender with an account and a username. Anyone else posts as a guest.
func (a Author) PostsAsPlayer(account AccountID) bool {
	return account != cpsession.NoAccount && a.Username != ""
}

// Reactions is who put which reaction on one message, in the order each reaction first appeared. It is a value:
// With and Without answer a changed copy and leave the receiver as it was.
type Reactions struct {
	given []given
}

type given struct {
	reaction Reaction
	reactors []Reactor
}

// Given is whether reactor has put reaction on.
func (r Reactions) Given(reaction Reaction, reactor Reactor) bool {
	index := r.index(reaction)
	return index >= 0 && slices.Contains(r.given[index].reactors, reactor)
}

// With is r with reaction put on by reactor. A reaction nobody gave yet goes last.
func (r Reactions) With(reaction Reaction, reactor Reactor) Reactions {
	if r.Given(reaction, reactor) {
		return r
	}

	next := r.clone()
	index := next.index(reaction)
	if index < 0 {
		next.given = append(next.given, given{reaction: reaction, reactors: []Reactor{reactor}})
		return next
	}

	next.given[index].reactors = append(slices.Clone(next.given[index].reactors), reactor)
	return next
}

// Without is r with reaction taken off by reactor. A reaction nobody gives any more is gone, and its place with
// it: given again, it goes last.
func (r Reactions) Without(reaction Reaction, reactor Reactor) Reactions {
	if !r.Given(reaction, reactor) {
		return r
	}

	next := r.clone()
	index := next.index(reaction)
	reactors := slices.DeleteFunc(slices.Clone(next.given[index].reactors), func(each Reactor) bool {
		return each == reactor
	})
	if len(reactors) == 0 {
		next.given = slices.Delete(next.given, index, index+1)
		return next
	}

	next.given[index].reactors = reactors
	return next
}

// Tally is how many gave each reaction, in order, and whether viewer is one of them. NoReactor is one of none.
func (r Reactions) Tally(viewer Reactor) []Count {
	counts := make([]Count, 0, len(r.given))
	for _, each := range r.given {
		counts = append(counts, Count{
			Reaction: each.reaction,
			Count:    len(each.reactors),
			Mine:     viewer != NoReactor && slices.Contains(each.reactors, viewer),
		})
	}
	return counts
}

func (r Reactions) index(reaction Reaction) int {
	return slices.IndexFunc(r.given, func(each given) bool { return each.reaction == reaction })
}

// clone copies the outer slice only: a changed entry gets its own reactors slice before it is written.
func (r Reactions) clone() Reactions {
	return Reactions{given: slices.Clone(r.given)}
}

// Count is how many put one reaction on one message.
type Count struct {
	Reaction Reaction
	Count    int
	Mine     bool
}

// Tally is a message's reactions as the stream sends them: for nobody in particular.
type Tally struct {
	MessageID MessageID
	Counts    []Count
}

// ReactionChange is one reactor putting one reaction on one message, or taking it off.
type ReactionChange struct {
	MessageID MessageID
	Reaction  Reaction
	Reactor   Reactor
	On        bool
	At        time.Time
}

// Applied is r with the change made.
func (r Reactions) Applied(change ReactionChange) Reactions {
	if change.On {
		return r.With(change.Reaction, change.Reactor)
	}
	return r.Without(change.Reaction, change.Reactor)
}

// ErrUnknownMessage is a reaction to a message the chat no longer shows, or never did.
var ErrUnknownMessage = errors.New("no such chat message")

// ErrInvalidReaction is a reaction the proto does not name.
var ErrInvalidReaction = errors.New("invalid reaction")
