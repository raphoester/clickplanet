package notices

import (
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Spread is the tiles a spread click took, told to every stream at once.
type Spread struct {
	CountryID string
	Tile      uint32

	Neighbours []uint32
}

// Event is one notice: a spread, or all a signed-in account holds after a grant.
type Event struct {
	Spread  *Spread
	Granted *charges.Held
}

const buffer = 32

// Board tells the streams open what planet's own wallet and clicks did beside the map's changes: a spread to every
// stream, a grant to the signed-in account's own. A stream that is not reading is dropped from, never waited for.
type Board struct {
	mu      sync.Mutex
	streams map[uint64]stream
	nextID  uint64
}

type stream struct {
	caller clicks.Caller
	events chan Event
}

func New() *Board {
	return &Board{streams: make(map[uint64]stream)}
}

func (b *Board) Attend(caller clicks.Caller) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	events := make(chan Event, buffer)
	b.streams[id] = stream{caller: caller, events: events}

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

// A guest's streams share its address with others', so only a signed-in account hears what it holds.
func (b *Board) PublishGranted(holder charges.Holder, held charges.Held) {
	account := clicks.CallerOf(clicks.Payer{Account: string(holder), Linked: true})
	b.send(func(each stream) bool { return each.caller == account }, Event{Granted: &held})
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
