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

type Reaction int32

type Reactor string

const NoReactor Reactor = ""

const accountPrefix = "account:"

func ReactorOf(account messages.AccountID) Reactor {
	if account == cpsession.NoAccount {
		return NoReactor
	}
	return Reactor(accountPrefix + account.String())
}

func AccountOf(reactor Reactor) (messages.AccountID, bool) {
	id, found := strings.CutPrefix(string(reactor), accountPrefix)
	if !found {
		return cpsession.NoAccount, false
	}

	account := messages.AccountIDOf(id)
	return account, account != cpsession.NoAccount
}

type Reactions struct {
	given   []given
	version uint64
}

type given struct {
	reaction Reaction
	reactors []Reactor
}

func (r Reactions) Given(reaction Reaction, reactor Reactor) bool {
	index := r.index(reaction)
	return index >= 0 && slices.Contains(r.given[index].reactors, reactor)
}

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

// Shallow: give an entry its own reactors slice before writing to it.
func (r Reactions) clone() Reactions {
	return Reactions{given: slices.Clone(r.given), version: r.version}
}

func (r Reactions) Version() uint64 {
	return r.version
}

func (r Reactions) Versioned(version uint64) Reactions {
	next := r.clone()
	next.version = version
	return next
}

func (r Reactions) TallyOf(id messages.MessageID) Tally {
	return Tally{MessageID: id, Counts: r.Tally(NoReactor), Version: r.version}
}

type Count struct {
	Reaction Reaction
	Count    int
	Mine     bool
	Reactors []Reactor
	Names    []string
}

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

type Tally struct {
	MessageID messages.MessageID
	Counts    []Count
	Version   uint64
}

type Change struct {
	MessageID messages.MessageID
	Reaction  Reaction
	Reactor   Reactor
	On        bool
	At        time.Time
}

func (r Reactions) Applied(change Change) Reactions {
	if change.On {
		return r.With(change.Reaction, change.Reactor)
	}
	return r.Without(change.Reaction, change.Reactor)
}

var ErrUnknownMessage = errors.New("no such chat message")

var ErrInvalidReaction = errors.New("invalid reaction")

type Storage interface {
	Save(ctx context.Context, change Change) error
	Reactions(ctx context.Context, ids []messages.MessageID) (map[messages.MessageID]Reactions, error)
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
