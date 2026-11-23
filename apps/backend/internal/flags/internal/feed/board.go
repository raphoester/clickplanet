package feed

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
)

// News is what an open stream hears: a fact decided while this process runs. The facts told again at its boot were
// decided before, and a page reads the list as its stream opens.
type News struct {
	since time.Time
}

func NewsSince(boot time.Time) News {
	return News{since: boot}
}

func (n News) Fresh(decidedAt time.Time) bool {
	return !decidedAt.Before(n.since)
}

// Board hands each flag opened or closed to every stream open. A stream that falls its buffer behind is closed, not
// skipped: its page opens it again and reads the list, where a skipped flag would stay unknown to it.
type Board struct {
	mu      sync.Mutex
	streams map[uint64]chan catalog.Snapshot
	nextID  uint64
	buffer  int
}

func NewBoard(buffer int) *Board {
	return &Board{streams: map[uint64]chan catalog.Snapshot{}, buffer: buffer}
}

func (b *Board) Attend() (<-chan catalog.Snapshot, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := b.nextID
	flags := make(chan catalog.Snapshot, b.buffer)
	b.streams[id] = flags
	return flags, func() { b.leave(id) }
}

func (b *Board) leave(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if flags, open := b.streams[id]; open {
		close(flags)
		delete(b.streams, id)
	}
}

func (b *Board) Publish(flag catalog.Snapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, flags := range b.streams {
		select {
		case flags <- flag:
		default:
			close(flags)
			delete(b.streams, id)
		}
	}
}
