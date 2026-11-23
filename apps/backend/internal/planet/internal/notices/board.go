package notices

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
)

// Spread is the tiles a spread click took, told to every stream at once.
type Spread struct {
	CountryID string
	Tile      uint32

	Neighbours []uint32
}

// MineLaid is a mine a player of the flag laid, told to the streams of that flag alone: its tile holds it while it has
// not changed since Version.
type MineLaid struct {
	Flag    string
	Tile    uint32
	Version clicks.MapVersion
}

// RallyCalled is a rally a player called, told to every stream at once with the time it has left.
type RallyCalled struct {
	Rally rallies.Snapshot
	Left  time.Duration
}

// Event is one notice: a spread, what a grant added to an account's hand and all it holds after, a mine laid, or a
// rally called.
type Event struct {
	Spread   *Spread
	Granted  *charges.Gain
	MineLaid *MineLaid
	Rally    *RallyCalled
}

const buffer = 32

// Board tells the streams open what planet's own wallet and clicks did beside the map's changes: a spread to every
// stream, a grant to the granted account's own, a mine laid to the streams of its flag's players. A stream that is not
// reading is dropped from, never waited for.
type Board struct {
	mu      sync.Mutex
	streams map[uint64]stream
	nextID  uint64
}

type stream struct {
	holder charges.Holder
	flag   string
	events chan Event
}

func New() *Board {
	return &Board{streams: make(map[uint64]stream)}
}

// Attend opens a stream for the account its session names, which plays for the flag: NoHolder for one with none, which
// hears no grant, and no flag for one that plays for none, which hears no mine.
func (b *Board) Attend(holder charges.Holder, flag string) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	events := make(chan Event, buffer)
	b.streams[id] = stream{holder: holder, flag: flag, events: events}

	return events, func() { b.leave(id) }
}

func (b *Board) leave(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.streams, id)
}

// Pledged has every stream of the account hear the mines of the flag it plays for now, and no more the ones of the flag
// before.
func (b *Board) Pledged(holder charges.Holder, flag string) {
	if holder == charges.NoHolder {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for id, each := range b.streams {
		if each.holder == holder {
			each.flag = flag
			b.streams[id] = each
		}
	}
}

func (b *Board) PublishSpread(spread Spread) {
	b.send(func(stream) bool { return true }, Event{Spread: &spread})
}

// A grant reaches only the streams whose session names the account granted, a guest's as a signed-in one's: a guest's
// token names its own account, which the others behind its address do not share.
func (b *Board) PublishGranted(holder charges.Holder, gain charges.Gain) {
	if holder == charges.NoHolder {
		return
	}
	b.send(func(each stream) bool { return each.holder == holder }, Event{Granted: &gain})
}

// A rally reaches every stream: every flag sees the ring.
func (b *Board) PublishRally(called RallyCalled) {
	b.send(func(stream) bool { return true }, Event{Rally: &called})
}

// A mine laid reaches only the streams of the players of its flag: the others never learn of it.
func (b *Board) PublishMineLaid(laid MineLaid) {
	if laid.Flag == "" {
		return
	}
	b.send(func(each stream) bool { return each.flag == laid.Flag }, Event{MineLaid: &laid})
}

func (b *Board) send(to func(stream) bool, event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, each := range b.streams {
		if !to(each) {
			continue
		}
		select {
		case each.events <- event:
		default:
		}
	}
}
