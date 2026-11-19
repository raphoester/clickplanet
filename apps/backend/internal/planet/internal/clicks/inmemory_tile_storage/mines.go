package inmemory_tile_storage

import (
	"context"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// The mines laid on the map in play, the ones a take set off and no explosion took yet, and the tiles whose mine changed
// since the last flush. A mine set off is no longer on the map: a stop before its explosion loses it.
type mineBook struct {
	laid    map[uint32]clicks.Mine
	tripped []clicks.Mine
	unsaved *cpcolls.Set[uint32]
}

func newMineBook() *mineBook {
	return &mineBook{laid: map[uint32]clicks.Mine{}, unsaved: cpcolls.NewSet[uint32]()}
}

func (b *mineBook) has(tile uint32) bool {
	_, ok := b.laid[tile]
	return ok
}

func (b *mineBook) lay(mine clicks.Mine) {
	tile := mine.TakeSnapshot().Tile
	b.laid[tile] = mine
	b.unsaved.Add(tile)
}

// A stored mine put back at boot: nothing to flush.
func (b *mineBook) restore(kept clicks.MineSnapshot) {
	b.laid[kept.Tile] = kept.Restore()
}

// take takes the tile's mine off the map; false for a tile with none.
func (b *mineBook) take(tile uint32) (clicks.Mine, bool) {
	mine, ok := b.laid[tile]
	if !ok {
		return clicks.Mine{}, false
	}
	delete(b.laid, tile)
	b.unsaved.Add(tile)

	return mine, true
}

// trip moves the tile's mine to the ones set off; false for a tile with none.
func (b *mineBook) trip(tile uint32) bool {
	mine, ok := b.take(tile)
	if ok {
		b.tripped = append(b.tripped, mine)
	}
	return ok
}

func (b *mineBook) takeTripped() (clicks.Mine, bool) {
	if len(b.tripped) == 0 {
		return clicks.Mine{}, false
	}
	mine := b.tripped[0]
	b.tripped = b.tripped[1:]

	return mine, true
}

// forget takes every mine of a deleted account off the map, with no blast: none goes off for nobody.
func (b *mineBook) forget(account cpsession.AccountID) {
	for tile, mine := range b.laid {
		if mine.TakeSnapshot().Account == account {
			b.take(tile)
		}
	}
	b.tripped = slices.DeleteFunc(b.tripped, func(mine clicks.Mine) bool { return mine.TakeSnapshot().Account == account })
}

// dropTripped forgets the mines of a flag knocked out that a take set off: its land is blank, and none of it comes back.
func (b *mineBook) dropTripped(flag string) {
	b.tripped = slices.DeleteFunc(b.tripped, func(mine clicks.Mine) bool { return mine.TakeSnapshot().Flag == flag })
}

func (b *mineBook) of(flag string) []uint32 {
	var tiles []uint32
	for tile, mine := range b.laid {
		if mine.TakeSnapshot().Flag == flag {
			tiles = append(tiles, tile)
		}
	}
	slices.Sort(tiles)
	return tiles
}

// drainUnsaved is each mine laid and each tile whose mine left since the last flush.
func (b *mineBook) drainUnsaved() Mines {
	tiles := make([]uint32, 0, b.unsaved.Len())
	b.unsaved.ForEach(func(tile uint32) { tiles = append(tiles, tile) })
	slices.Sort(tiles)
	b.unsaved.Clear()

	var mines Mines
	for _, tile := range tiles {
		if mine, ok := b.laid[tile]; ok {
			mines.Laid = append(mines.Laid, mine.TakeSnapshot())
			continue
		}
		mines.Gone = append(mines.Gone, tile)
	}

	return mines
}

func (b *mineBook) markUnsaved(mines Mines) {
	for _, mine := range mines.Laid {
		b.unsaved.Add(mine.Tile)
	}
	b.unsaved.Add(mines.Gone...)
}

// Mines is what changed among a map's mines: each one laid to keep, and each tile whose mine is to delete.
type Mines struct {
	Laid []clicks.MineSnapshot
	Gone []uint32
}

func (m Mines) empty() bool {
	return len(m.Laid) == 0 && len(m.Gone) == 0
}

// Fuse has a value once a take set a mine off, until Explode takes it.
func (s *Storage) Fuse() <-chan struct{} {
	return s.fuse
}

// tripLocked sets off the tile's mine, and lights the fuse.
func (s *Storage) tripLocked(tile uint32) {
	if !s.mines.trip(tile) {
		return
	}
	select {
	case s.fuse <- struct{}{}:
	default:
	}
}

func (s *Storage) MinesOf(flag string) ([]uint32, clicks.MapVersion) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.mines.of(flag), s.version
}

// Lay holds the lock through `write`, and answers the map's version under it: a lay changes no tile, and is told to no
// stream of the map's.
func (s *Storage) Lay(_ context.Context, write func(tiles clicks.Laying) error) (clicks.MapVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := write(&insideLaying{storage: s})

	return s.version, err
}

type insideLaying struct {
	storage *Storage
}

var _ clicks.Laying = (*insideLaying)(nil)

func (l *insideLaying) Tile(id uint32) (clicks.Tile, error) {
	return l.storage.tileLocked(id)
}

func (l *insideLaying) Mined(tile uint32) bool {
	return l.storage.mines.has(tile)
}

func (l *insideLaying) Put(mine clicks.Mine) {
	l.storage.mines.lay(mine)
}

// Explode holds the lock through every write of `write`: one change at one version with every blast, then the fortifies
// they earned, then the knock-outs, then the cut-offs.
func (s *Storage) Explode(_ context.Context, write func(tiles clicks.Exploding) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inside := &insideExplosion{storage: s}
	err := write(inside)
	if len(inside.blasts) > 0 {
		s.publishLocked(clicks.Change{Explosion: &clicks.Explosion{Blasts: inside.blasts}})
	}
	for i := range inside.fortified {
		s.publishLocked(clicks.Change{Fortification: &inside.fortified[i]})
	}
	for i := range inside.knockedOut {
		s.publishLocked(clicks.Change{KnockOut: &inside.knockedOut[i]})
	}
	for i := range inside.cutOff {
		s.publishLocked(clicks.Change{CutOff: &inside.cutOff[i]})
	}

	return err
}

type insideExplosion struct {
	storage    *Storage
	blasts     []clicks.MineBlast
	fortified  []clicks.Fortification
	knockedOut []clicks.KnockOut
	cutOff     []clicks.CutOff
}

var _ clicks.Exploding = (*insideExplosion)(nil)

func (e *insideExplosion) Tile(id uint32) (clicks.Tile, error) {
	return e.storage.tileLocked(id)
}

func (e *insideExplosion) Set(_ context.Context, tile clicks.Tile) error {
	_, _, err := e.storage.writeLocked(tile, false)
	return err
}

func (e *insideExplosion) Click(ctx context.Context, tile clicks.Tile) error {
	return e.Set(ctx, tile)
}

func (e *insideExplosion) Fortify(ctx context.Context, tile uint32, flag string, most int) (clicks.Fortification, error) {
	fortification, err := e.storage.fortifyLocked(ctx, tile, flag, most)
	if err != nil {
		return clicks.Fortification{}, err
	}
	e.fortified = append(e.fortified, fortification)

	return fortification, nil
}

func (e *insideExplosion) KnockOut(ctx context.Context, tile uint32, flag string) (clicks.KnockOut, error) {
	knockOut, err := e.storage.knockOutLocked(ctx, tile, flag)
	if err != nil {
		return clicks.KnockOut{}, err
	}
	e.knockedOut = append(e.knockedOut, knockOut)

	return knockOut, nil
}

func (e *insideExplosion) Trip(tile uint32) {
	e.storage.tripLocked(tile)
}

func (e *insideExplosion) TakeTripped() (clicks.Mine, bool) {
	return e.storage.mines.takeTripped()
}

func (e *insideExplosion) SetOff(tile uint32) (clicks.Mine, bool) {
	return e.storage.mines.take(tile)
}

func (e *insideExplosion) Blasted(blast clicks.MineBlast) {
	e.blasts = append(e.blasts, blast)
}

func (e *insideExplosion) CutOff(losses clicks.Losses) (clicks.CutOff, error) {
	cutOff, err := e.storage.cutOffLocked(losses)
	if err != nil {
		return clicks.CutOff{}, err
	}
	e.cutOff = append(e.cutOff, cutOff)

	return cutOff, nil
}
