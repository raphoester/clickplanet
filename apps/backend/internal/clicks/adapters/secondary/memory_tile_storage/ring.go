package memory_tile_storage

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
)

type timedUpdate struct {
	at     time.Time
	update domain.TileUpdate
}

type ring struct {
	buf   []timedUpdate
	start int
	size  int
}

func newRing(capacity int) *ring {
	return &ring{buf: make([]timedUpdate, capacity)}
}

func (r *ring) push(entry timedUpdate) {
	if len(r.buf) == 0 {
		return
	}

	if r.size == len(r.buf) {
		r.buf[r.start] = entry
		r.start = (r.start + 1) % len(r.buf)
		return
	}

	r.buf[(r.start+r.size)%len(r.buf)] = entry
	r.size++
}

func (r *ring) evictBefore(cutoff time.Time) {
	for r.size > 0 && r.buf[r.start].at.Before(cutoff) {
		r.buf[r.start] = timedUpdate{}
		r.start = (r.start + 1) % len(r.buf)
		r.size--
	}
}

func (r *ring) since(start time.Time) []domain.TileUpdate {
	updates := make([]domain.TileUpdate, 0, r.size)
	for i := 0; i < r.size; i++ {
		entry := r.buf[(r.start+i)%len(r.buf)]
		if entry.at.Before(start) {
			continue
		}
		updates = append(updates, entry.update)
	}
	return updates
}
