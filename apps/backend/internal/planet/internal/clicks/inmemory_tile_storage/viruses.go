package inmemory_tile_storage

import (
	"cmp"
	"context"
	"maps"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The live viruses of the map in play and the tile each infects, kept with the map's tiles and saved by the same flush.
type virusBook struct {
	live     map[clicks.VirusID]clicks.Virus
	infected map[uint32]clicks.VirusID
	unsaved  bool
}

func newVirusBook() *virusBook {
	return &virusBook{live: map[clicks.VirusID]clicks.Virus{}, infected: map[uint32]clicks.VirusID{}}
}

// keep holds the virus as it is: one that ended leaves the book, and its tiles are infected no more.
func (b *virusBook) keep(virus clicks.Virus) {
	kept := virus.TakeSnapshot()
	b.forget(kept.ID)
	b.unsaved = true
	if kept.Ended {
		return
	}
	b.put(virus)
}

// A stored virus put back at boot: nothing to flush.
func (b *virusBook) restore(kept clicks.VirusSnapshot) {
	b.put(kept.Restore())
}

func (b *virusBook) put(virus clicks.Virus) {
	kept := virus.TakeSnapshot()
	b.live[kept.ID] = virus
	for _, tile := range kept.Infected {
		b.infected[tile] = kept.ID
	}
}

func (b *virusBook) forget(id clicks.VirusID) {
	virus, ok := b.live[id]
	if !ok {
		return
	}
	for _, tile := range virus.TakeSnapshot().Infected {
		delete(b.infected, tile)
	}
	delete(b.live, id)
}

// cure takes the tile out of the virus that infects it: the tile changed hands.
func (b *virusBook) cure(tile uint32) {
	id, ok := b.infected[tile]
	if !ok {
		return
	}
	delete(b.infected, tile)
	b.live[id] = b.live[id].Cured(tile)
	b.unsaved = true
}

func (b *virusBook) virus(id clicks.VirusID) (clicks.Virus, bool) {
	virus, ok := b.live[id]
	return virus, ok
}

func (b *virusBook) all() []clicks.Virus {
	ids := slices.Sorted(maps.Keys(b.live))
	viruses := make([]clicks.Virus, len(ids))
	for i, id := range ids {
		viruses[i] = b.live[id]
	}
	return viruses
}

func (b *virusBook) infectedIn(start, end uint32) []clicks.InfectedTile {
	var tiles []clicks.InfectedTile
	for tile, id := range b.infected {
		if tile >= start && tile <= end {
			tiles = append(tiles, clicks.InfectedTile{Tile: tile, Virus: id})
		}
	}
	slices.SortFunc(tiles, func(a, b clicks.InfectedTile) int { return cmp.Compare(a.Tile, b.Tile) })
	return tiles
}

// drainUnsaved is every live virus when one changed since the last flush: the map's viruses are kept as a whole.
func (b *virusBook) drainUnsaved() Viruses {
	if !b.unsaved {
		return Viruses{}
	}
	b.unsaved = false

	live := make([]clicks.VirusSnapshot, 0, len(b.live))
	for _, virus := range b.all() {
		live = append(live, virus.TakeSnapshot())
	}
	return Viruses{Changed: true, Live: live}
}

func (b *virusBook) markUnsaved(viruses Viruses) {
	b.unsaved = b.unsaved || viruses.Changed
}

// Viruses is every live virus of a map, to keep in place of the ones kept before, when Changed.
type Viruses struct {
	Changed bool
	Live    []clicks.VirusSnapshot
}

func (s *Storage) Viruses() []clicks.Virus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.viruses.all()
}

// Infect holds the lock through every write of `write`: one change at one version with what it took and struck and the
// virus after it, then the fortifies it earned, then the knock-outs.
func (s *Storage) Infect(_ context.Context, write func(tiles clicks.Infecting) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inside := &insideInfection{storage: s}
	err := write(inside)
	if inside.kept != nil {
		s.publishLocked(clicks.Change{Infection: &clicks.Infection{
			Virus: inside.kept.TakeSnapshot(), Taken: inside.taken, Struck: inside.struck,
		}})
	}
	for i := range inside.fortified {
		s.publishLocked(clicks.Change{Fortification: &inside.fortified[i]})
	}
	for i := range inside.knockedOut {
		s.publishLocked(clicks.Change{KnockOut: &inside.knockedOut[i]})
	}

	return err
}

type insideInfection struct {
	storage    *Storage
	kept       *clicks.Virus
	taken      []uint32
	struck     []clicks.TileShields
	fortified  []clicks.Fortification
	knockedOut []clicks.KnockOut
}

var _ clicks.Infecting = (*insideInfection)(nil)

func (i *insideInfection) Tile(id uint32) (clicks.Tile, error) {
	return i.storage.tileLocked(id)
}

func (i *insideInfection) Set(_ context.Context, tile clicks.Tile) error {
	return i.put(tile)
}

func (i *insideInfection) Click(_ context.Context, tile clicks.Tile) error {
	return i.put(tile)
}

func (i *insideInfection) put(tile clicks.Tile) error {
	update, changed, err := i.storage.writeLocked(tile, false)
	if !changed {
		return err
	}
	if update.Value == update.Previous {
		i.struck = append(i.struck, clicks.TileShields{Tile: update.Tile, Shields: update.Shields})
	} else {
		i.taken = append(i.taken, update.Tile)
	}

	return err
}

func (i *insideInfection) Trip(tile uint32) {
	i.storage.tripLocked(tile)
}

func (i *insideInfection) Fortify(ctx context.Context, tile uint32, flag string, most int) (clicks.Fortification, error) {
	fortification, err := i.storage.fortifyLocked(ctx, tile, flag, most)
	if err != nil {
		return clicks.Fortification{}, err
	}
	i.fortified = append(i.fortified, fortification)

	return fortification, nil
}

func (i *insideInfection) KnockOut(ctx context.Context, tile uint32, flag string) (clicks.KnockOut, error) {
	knockOut, err := i.storage.knockOutLocked(ctx, tile, flag)
	if err != nil {
		return clicks.KnockOut{}, err
	}
	i.knockedOut = append(i.knockedOut, knockOut)

	return knockOut, nil
}

func (i *insideInfection) Virus(id clicks.VirusID) (clicks.Virus, bool) {
	return i.storage.viruses.virus(id)
}

func (i *insideInfection) Keep(virus clicks.Virus) {
	i.storage.viruses.keep(virus)
	i.kept = &virus
}
