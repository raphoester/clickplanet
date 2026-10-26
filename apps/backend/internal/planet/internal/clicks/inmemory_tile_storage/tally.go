package inmemory_tile_storage

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const tallyInterval = time.Second

func (s *Storage) Tally() (clicks.MapVersion, clicks.Tally) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tally := clicks.Tally{Countries: s.board.counts()}
	s.landmasses.each(func(landmass clicks.LandmassID) {
		if row := s.landmasses.row(landmass, s.board.nameOf); len(row.Flags) > 0 {
			tally.Landmasses = append(tally.Landmasses, row)
		}
	})

	return s.version, tally
}

// Publishes the rows the writes changed since it last ran, with no new version: it reads, it writes nothing.
func (s *Storage) PublishTally() {
	s.mu.Lock()
	if tally, changed := s.recountLocked(); changed {
		s.feed.publish(clicks.Change{Version: s.version, Tally: &tally})
	}
	s.mu.Unlock()

	s.feed.reportCut()
}

func (s *Storage) recountLocked() (clicks.Tally, bool) {
	var tally clicks.Tally
	s.recount.flags.drain(func(code uint32) {
		if code != uint32(unownedCode) {
			tally.Countries = append(tally.Countries, s.board.countOf(uint16(code))) //nolint:gosec // a code the set was marked with.
		}
	})
	s.recount.landmasses.drain(func(id uint32) {
		if landmass := clicks.LandmassID(id); landmass != clicks.NoLandmass { //nolint:gosec // landmasses are uint16 in the blob.
			tally.Landmasses = append(tally.Landmasses, s.landmasses.row(landmass, s.board.nameOf))
		}
	})

	return tally, len(tally.Countries) > 0 || len(tally.Landmasses) > 0
}

func (s *Storage) publishTallyEvery(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.PublishTally()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Storage) Version() clicks.MapVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.version
}
