package inmemory_tile_storage

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const censusInterval = time.Second

func (s *Storage) Census() (clicks.MapVersion, clicks.Census) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	census := clicks.Census{Countries: s.board.counts()}
	s.landmasses.each(func(landmass clicks.LandmassID) {
		if row := s.landmasses.row(landmass, s.board.nameOf); len(row.Flags) > 0 {
			census.Landmasses = append(census.Landmasses, row)
		}
	})

	return s.version, census
}

// Publishes the rows the writes changed since it last ran, with no new version: it reads, it writes nothing.
func (s *Storage) PublishCensus() {
	s.mu.Lock()
	if census, changed := s.recountLocked(); changed {
		s.feed.publish(clicks.Change{Version: s.version, Census: &census})
	}
	s.mu.Unlock()

	s.feed.reportCut()
}

func (s *Storage) recountLocked() (clicks.Census, bool) {
	var census clicks.Census
	s.recount.flags.drain(func(code uint32) {
		if code != uint32(unownedCode) {
			census.Countries = append(census.Countries, s.board.countOf(uint16(code))) //nolint:gosec // a code the set was marked with.
		}
	})
	s.recount.landmasses.drain(func(id uint32) {
		if landmass := clicks.LandmassID(id); landmass != clicks.NoLandmass { //nolint:gosec // landmasses are uint16 in the blob.
			census.Landmasses = append(census.Landmasses, s.landmasses.row(landmass, s.board.nameOf))
		}
	})

	return census, len(census.Countries) > 0 || len(census.Landmasses) > 0
}

func (s *Storage) publishCensusEvery(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.PublishCensus()
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
