// Package reactions is what people put on a chat message: which reaction, from whom, and where it is kept.
package reactions

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Reaction is one of chat.v1.Reaction, by its wire number. The number is what is stored, which is why the proto
// never renumbers one. The edge refuses a number the proto does not name.
type Reaction int32

// Reactor is who put a reaction on: an account. Before guests had accounts a guest reacted as the tag of its
// address, "guest:" and the tag; those rows age out with the retention, and no caller matches them any more.
type Reactor string

// NoReactor is nobody: what the stream tallies for, since it cannot know who reads it, and a caller with no
// account.
const NoReactor Reactor = ""

const accountPrefix = "account:"

// ReactorOf is who reacts: the account, whether it chose a username or not. cpsession.NoAccount is NoReactor.
func ReactorOf(account messages.AccountID) Reactor {
	if account == cpsession.NoAccount {
		return NoReactor
	}
	return Reactor(accountPrefix + account.String())
}

// AccountOf is the account a reactor stands for, and whether it is one at all: a "guest:" row from before
// guests had accounts is nobody, and nobody can be named.
func AccountOf(reactor Reactor) (messages.AccountID, bool) {
	id, found := strings.CutPrefix(string(reactor), accountPrefix)
	if !found {
		return cpsession.NoAccount, false
	}

	account := messages.AccountIDOf(id)
	return account, account != cpsession.NoAccount
}

// Reactions is who put which reaction on one message, in the order each reaction first appeared. It is a value:
// With and Without answer a changed copy and leave the receiver as it was.
type Reactions struct {
	given   []given
	version uint64
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

// Tally is how many gave each reaction, in order, who they are and whether viewer is one of them. NoReactor is
// one of none. The reactors are accounts: Named turns them into what a reader sees.
func (r Reactions) Tally(viewer Reactor) []Count {
	counts := make([]Count, 0, len(r.given))
	for _, each := range r.given {
		counts = append(counts, Count{
			Reaction: each.reaction,
			Count:    len(each.reactors),
			Mine:     viewer != NoReactor && slices.Contains(each.reactors, viewer),
			Reactors: slices.Clone(each.reactors),
		})
	}
	return counts
}

func (r Reactions) index(reaction Reaction) int {
	return slices.IndexFunc(r.given, func(each given) bool { return each.reaction == reaction })
}

// clone copies the outer slice only: a changed entry gets its own reactors slice before it is written.
func (r Reactions) clone() Reactions {
	return Reactions{given: slices.Clone(r.given), version: r.version}
}

// Version is which state of the message's reactions this is: the store bumps it with each change, never here.
func (r Reactions) Version() uint64 {
	return r.version
}

// Versioned is r as the store read it at version.
func (r Reactions) Versioned(version uint64) Reactions {
	next := r.clone()
	next.version = version
	return next
}

// TallyOf is what the stream sends for the message: its counts for nobody, and their version.
func (r Reactions) TallyOf(id messages.MessageID) Tally {
	return Tally{MessageID: id, Counts: r.Tally(NoReactor), Version: r.version}
}

// Count is how many put one reaction on one message, and who.
//
// It carries the same two halves a messages.Message does: Reactors is what is stored, Names is who those
// accounts are to a reader, filled by Named when the count is about to be shown.
type Count struct {
	Reaction Reaction
	Count    int
	Mine     bool
	// Reactors is who gave it, oldest first, as accounts.
	Reactors []Reactor
	// Names is those same people as a reader sees them now, oldest first. Shorter than Count when somebody
	// cannot be named any more, so Count is what says how many gave it.
	Names []string
}

// Named is counts with everyone in them named by who their account is now, so a rename shows under every
// reaction its player ever gave. Somebody nobody can name any more — a deleted account, or a guest from
// before guests had one — is left out rather than stood in for: a list of reactions is not the place to say
// somebody is gone, and the count already says how many there are.
func Named(counts []Count, authors map[messages.AccountID]messages.Author) []Count {
	named := make([]Count, 0, len(counts))
	for _, count := range counts {
		count.Names = namesOf(count.Reactors, authors)
		named = append(named, count)
	}
	return named
}

func namesOf(reactors []Reactor, authors map[messages.AccountID]messages.Author) []string {
	names := make([]string, 0, len(reactors))
	for _, reactor := range reactors {
		account, isAccount := AccountOf(reactor)
		if !isAccount {
			continue
		}
		if author, known := authors[account]; known {
			names = append(names, author.Name)
		}
	}
	return names
}

// AccountsOf is every account under these counts, each once, for one batch ask.
func AccountsOf(counts []Count) []messages.AccountID {
	seen := cpcolls.NewSetWithCapacity[messages.AccountID](len(counts))
	accounts := make([]messages.AccountID, 0, len(counts))
	for _, count := range counts {
		for _, reactor := range count.Reactors {
			account, isAccount := AccountOf(reactor)
			if !isAccount || seen.Contains(account) {
				continue
			}
			seen.Add(account)
			accounts = append(accounts, account)
		}
	}
	return accounts
}

// Tally is a message's reactions as the stream sends them: for nobody in particular.
type Tally struct {
	MessageID messages.MessageID
	Counts    []Count
	Version   uint64
}

// Change is one reactor putting one reaction on one message, or taking it off.
type Change struct {
	MessageID messages.MessageID
	Reaction  Reaction
	Reactor   Reactor
	On        bool
	At        time.Time
}

// Applied is r with the change made.
func (r Reactions) Applied(change Change) Reactions {
	if change.On {
		return r.With(change.Reaction, change.Reactor)
	}
	return r.Without(change.Reaction, change.Reactor)
}

// ErrUnknownMessage is a reaction to a message the chat no longer shows, or never did.
var ErrUnknownMessage = errors.New("no such chat message")

// ErrInvalidReaction is a reaction the proto does not name.
var ErrInvalidReaction = errors.New("invalid reaction")

// Storage is where reactions are kept. StorageContractSuite pins what every adapter does.
type Storage interface {
	// Save puts the reaction on or takes it off, and bumps the message's version in the same write when that
	// changed something. Either one already done is not an error, and bumps nothing.
	Save(ctx context.Context, change Change) error
	// Reactions is what each of the given messages carries, versioned. A message never reacted to is absent.
	Reactions(ctx context.Context, ids []messages.MessageID) (map[messages.MessageID]Reactions, error)
	// DeleteBefore removes every reaction put on before cutoff and says how many.
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
