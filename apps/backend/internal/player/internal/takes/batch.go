package takes

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var ErrDisordered = errors.New("the takes are not in position order inside what was read")

type Batch struct {
	from  Position
	next  Position
	takes []Take
}

func BatchOf(from, next Position, takes []Take) (Batch, error) {
	if next < from {
		return Batch{}, fmt.Errorf("%w: read up to %d from %d", ErrDisordered, next, from)
	}
	after := from
	for _, take := range takes {
		if take.position < after || take.position >= next {
			return Batch{}, fmt.Errorf("%w: %d read from %d up to %d", ErrDisordered, take.position, after, next)
		}
		after = take.position + 1
	}
	return Batch{from: from, next: next, takes: slices.Clone(takes)}, nil
}

func (b Batch) From() Position { return b.from }

func (b Batch) Next() Position { return b.next }

func (b Batch) Empty() bool { return b.next == b.from }

func (b Batch) Len() int { return len(b.takes) }

func (b Batch) Accounts() []players.AccountID {
	seen := cpcolls.NewSet[players.AccountID]()
	var accounts []players.AccountID
	for _, take := range b.takes {
		if take.Countable() && !seen.Contains(take.account) {
			seen.Add(take.account)
			accounts = append(accounts, take.account)
		}
	}
	slices.SortFunc(accounts, func(a, b players.AccountID) int { return bytes.Compare(a[:], b[:]) })
	return accounts
}

func (b Batch) Tallied(current map[players.AccountID]players.Stats) []players.Stats {
	next := maps.Clone(current)
	if next == nil {
		next = map[players.AccountID]players.Stats{}
	}
	for _, take := range b.takes {
		if !take.Countable() {
			continue
		}
		stats, ok := next[take.account]
		if !ok {
			stats = players.NewStats(take.account)
		}
		next[take.account] = stats.WithTake(take.at)
	}

	accounts := b.Accounts()
	tallied := make([]players.Stats, len(accounts))
	for i, account := range accounts {
		tallied[i] = next[account]
	}
	return tallied
}
