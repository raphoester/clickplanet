package notices

import (
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

// Spread is the tiles a spread click took, told to every stream at once.
type Spread struct {
	CountryID string
	Tile      uint32

	Neighbours []uint32
}

// Event is one notice: a spread, or what a grant added to an account's hand and all it holds after.
type Event struct {
	Spread  *Spread
	Granted *charges.Gain
}

const buffer = 32

// Board tells the streams open what planet's own wallet and clicks did beside the map's changes: a spread to every
// stream, a grant to the granted account's own. A stream that is not reading is dropped from, never waited for.
type Board struct {
	mu      sync.Mutex
	streams map[uint64]stream
	nextID  uint64
}

type stream struct {
	holder charges.Holder
	events chan Event
}

func New() *Board {
	return &Board{streams: make(map[uint64]stream)}
}

// Attend opens a stream for the account its session names: NoHolder for one with none, which hears no grant.
func (b *Board) Attend(holder charges.Holder) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	events := make(chan Event, buffer)
	b.streams[id] = stream{holder: holder, events: events}

	return events, func() { b.leave(id) }
}

func (b *Board) leave(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.streams, id)
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
