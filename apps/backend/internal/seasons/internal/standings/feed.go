package standings

import (
	"errors"
	"fmt"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Position uint64

type Entry struct {
	position Position
	take     Take
	reverted bool
}

func EntryOf(position Position, take Take, reverted bool) Entry {
	return Entry{position: position, take: take, reverted: reverted}
}

func (e Entry) Position() Position { return e.position }

func (e Entry) Countable() bool {
	return e.take.Account != cpsession.NoAccount && e.take.Country != "" && !e.reverted
}

var ErrDisordered = errors.New("the takes are not in position order inside what was read")

type Batch struct {
	from    Position
	next    Position
	entries []Entry
}

func BatchOf(from, next Position, entries []Entry) (Batch, error) {
	if next < from {
		return Batch{}, fmt.Errorf("%w: read up to %d from %d", ErrDisordered, next, from)
	}
	after := from
	for _, entry := range entries {
		if entry.position < after || entry.position >= next {
			return Batch{}, fmt.Errorf("%w: %d read from %d up to %d", ErrDisordered, entry.position, after, next)
		}
		after = entry.position + 1
	}
	return Batch{from: from, next: next, entries: slices.Clone(entries)}, nil
}

func (b Batch) From() Position { return b.from }

func (b Batch) Next() Position { return b.next }

func (b Batch) Empty() bool { return b.next == b.from }

func (b Batch) Len() int { return len(b.entries) }

func (b Batch) Takes() []Take {
	var takes []Take
	for _, entry := range b.entries {
		if entry.Countable() {
			takes = append(takes, entry.take)
		}
	}
	return takes
}
