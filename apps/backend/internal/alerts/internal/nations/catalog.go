package nations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

var ErrNotSold = errors.New("not the code of a flag opened since the game shipped")

// What the alerts heard of a flag opened since the game shipped: the name the latest fact told, from the flags
// module's FlagOpened and FlagRenamed. A flag closed keeps it, and its role: its players move to another flag, and
// their role follows.
type Sold struct {
	flag    Flag
	name    string
	namedAt time.Time
}

// Named keeps a name told at or after the time of the one it holds: a rename and an opening come on two streams, in
// no order between them, and the restatement at each boot tells the name now at the time of the opening, which puts a
// rename this module missed right. True when the name changed.
func (s Sold) Named(name string, at time.Time) (Sold, bool) {
	if at.Before(s.namedAt) || name == "" {
		return s, false
	}
	changed := name != s.name
	s.name, s.namedAt = name, at
	return s, changed
}

type SoldSnapshot struct {
	Flag    Flag
	Name    string
	NamedAt time.Time
}

func (s Sold) TakeSnapshot() SoldSnapshot {
	return SoldSnapshot{Flag: s.flag, Name: s.name, NamedAt: s.namedAt}
}

func (s SoldSnapshot) Restore() (Sold, error) {
	if !cpflags.IsOpenedCode(string(s.Flag)) {
		return Sold{}, fmt.Errorf("%w: %q", ErrNotSold, s.Flag)
	}
	return Sold{flag: s.Flag, name: s.Name, namedAt: s.NamedAt}, nil
}

type Store interface {
	SoldFlags(ctx context.Context) ([]Sold, error)
	SaveSold(ctx context.Context, sold Sold) error
}

// Load reads every sold flag heard, once at boot.
func Load(ctx context.Context, store Store) (*Catalog, error) {
	kept, err := store.SoldFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the sold flags heard: %w", err)
	}
	sold := make(map[Flag]Sold, len(kept))
	for _, flag := range kept {
		sold[flag.flag] = flag
	}
	return &Catalog{store: store, sold: sold}, nil
}

// The flags the server has a role for, and the name each role wears. The flags the game ships with are never read
// from postgres, and the sold ones are held in memory, written through: every take of a batch asks it, and the module
// runs in one process.
type Catalog struct {
	store Store

	mu   sync.RWMutex
	sold map[Flag]Sold
}

// No flag for a code the game does not ship and no fact named.
func (c *Catalog) FlagOf(code string) (Flag, bool) {
	if cpflags.IsBuiltIn(code) {
		return Flag(code), true
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, heard := c.sold[Flag(code)]; heard {
		return Flag(code), true
	}
	return "", false
}

// A flag the game ships with is named in English by the library, a sold one by its holder and the operator.
func (c *Catalog) NameOf(flag Flag) string {
	if cpflags.IsBuiltIn(string(flag)) {
		return flag.Name()
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if sold, heard := c.sold[flag]; heard {
		return sold.name
	}
	return strings.ToUpper(string(flag))
}

// Name keeps a fact's name of a sold flag: true when the name its role wears changed. A flag the game ships with is
// named by the library.
func (c *Catalog) Name(ctx context.Context, code, name string, at time.Time) (Flag, bool, error) {
	if !cpflags.IsOpenedCode(code) {
		return "", false, fmt.Errorf("%w: %q", ErrNotSold, code)
	}
	flag := Flag(code)

	c.mu.Lock()
	defer c.mu.Unlock()
	kept, heard := c.sold[flag]
	if !heard {
		kept = Sold{flag: flag}
	}
	named, changed := kept.Named(name, at)
	if named == kept {
		return flag, false, nil
	}
	if err := c.store.SaveSold(ctx, named); err != nil {
		return "", false, fmt.Errorf("failed to keep the name of %s: %w", flag, err)
	}
	c.sold[flag] = named
	return flag, changed || !heard, nil
}
