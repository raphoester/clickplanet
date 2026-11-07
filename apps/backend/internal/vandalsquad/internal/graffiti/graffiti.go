package graffiti

import (
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

type Kind int

const (
	Swastika Kind = iota + 1
)

// A drawing on the map the squad flagged: who painted it, with which flag, and where.
type Graffiti struct {
	Account cpsession.AccountID
	Flag    string
	Kind    Kind
	Tiles   []drawings.TileID
	// When the squad first saw it, and when it last did.
	FlaggedAt time.Time
	SeenAt    time.Time
	// The page opened on it.
	Link string
}

// One drawing seen again: the same player's paint of the same flag, on some of the same tiles.
func (g Graffiti) sameAs(other Graffiti) bool {
	if g.Account != other.Account || g.Flag != other.Flag || g.Kind != other.Kind {
		return false
	}
	for _, tile := range g.Tiles {
		if slices.Contains(other.Tiles, tile) {
			return true
		}
	}
	return false
}

// The graffiti flagged since the server started, newest first, capacity at most: a restart forgets them, and the log
// keeps a line of each.
type Book struct {
	capacity int

	mu      sync.Mutex
	entries []Graffiti
}

func NewBook(capacity int) *Book {
	return &Book{capacity: capacity}
}

// Whether the drawing was flagged already.
func (b *Book) Flagged(graffiti Graffiti) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return slices.ContainsFunc(b.entries, graffiti.sameAs)
}

// A drawing flagged already keeps its first flagging and takes the tiles and the link it was seen with now; a new one
// goes first, and the oldest goes once the book is full.
func (b *Book) Keep(graffiti Graffiti) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if i := slices.IndexFunc(b.entries, graffiti.sameAs); i >= 0 {
		graffiti.FlaggedAt = b.entries[i].FlaggedAt
		b.entries[i] = graffiti
		return
	}
	b.entries = slices.Insert(b.entries, 0, graffiti)
	if len(b.entries) > b.capacity {
		b.entries = b.entries[:b.capacity]
	}
}

func (b *Book) Entries() []Graffiti {
	b.mu.Lock()
	defer b.mu.Unlock()

	return slices.Clone(b.entries)
}

func (k Kind) String() string {
	switch k {
	case Swastika:
		return "swastika"
	default:
		return "unknown"
	}
}
