package roles

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

var ErrNoRole = errors.New("the bot made no such role")

type Key string

const flagPrefix = "flag:"

// A role the bot makes in the server the first time one is needed, kept by its key. Its name is the Book's to read: a
// flag the game ships with keeps the name it was made with, and a sold flag's follows the flags module's.
type Role struct {
	Key   Key
	Color discord.Color
}

// The game names no other role: everybody in the server is pinged when a battle starts.
func Nation(flag nations.Flag) Role {
	return Role{Key: Key(flagPrefix + string(flag)), Color: ColorOf(flag)}
}

func (k Key) code() (string, bool) {
	return strings.CutPrefix(string(k), flagPrefix)
}

const (
	saturation = 0.62
	lightness  = 0.55
)

// One bright color per flag, the same on every boot: its hue drawn from its code. Discord draws a role with no color
// in gray, and the flags' own colors are mostly the same red, white and blue.
func ColorOf(flag nations.Flag) discord.Color {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(flag))
	hue := float64(hash.Sum32()%360) / 60

	chroma := (1 - math.Abs(2*lightness-1)) * saturation
	x := chroma * (1 - math.Abs(math.Mod(hue, 2)-1))
	var r, g, b float64
	switch int(hue) {
	case 0:
		r, g = chroma, x
	case 1:
		r, g = x, chroma
	case 2:
		g, b = chroma, x
	case 3:
		g, b = x, chroma
	case 4:
		r, b = x, chroma
	default:
		r, b = chroma, x
	}
	m := lightness - chroma/2
	channel := func(v float64) int { return int(math.Round((v + m) * 255)) }
	return discord.Color(channel(r)<<16 | channel(g)<<8 | channel(b))
}

// What the store keeps of a role the bot made: its id, the color it was last painted, and the name it was given. No
// name: a flag the game ships with, whose name never follows (a role made before the names were kept has none either).
type Kept struct {
	ID    discord.RoleID
	Color discord.Color
	Name  string
}

type Store interface {
	RoleOf(ctx context.Context, key Key) (Kept, error)
	SaveRole(ctx context.Context, key Key, kept Kept) error
	Keys(ctx context.Context) ([]Key, error)
	ForgetRole(ctx context.Context, key Key) error
}

type Maker interface {
	MakeRole(ctx context.Context, name string, color discord.Color) (discord.RoleID, error)
	PaintRole(ctx context.Context, role discord.RoleID, color discord.Color) error
	RenameRole(ctx context.Context, role discord.RoleID, name string) error
	DeleteRole(ctx context.Context, role discord.RoleID) error
}

// nations.Catalog: the flags the server has a role for, and their names.
type Names interface {
	FlagOf(code string) (nations.Flag, bool)
	NameOf(flag nations.Flag) string
}

func NewBook(store Store, maker Maker, names Names) *Book {
	return &Book{store: store, maker: maker, names: names}
}

// One process makes each role once: two subscribers that need the same new role wait on each other.
type Book struct {
	mu    sync.Mutex
	store Store
	maker Maker
	names Names
}

// The name a role wears, and whether a role kept under another follows it: a sold flag's, which the operator may
// rename. A flag the game ships with keeps the name it was made with, so a new name in the library changes nothing.
func (b *Book) nameOf(role Role) (string, bool) {
	code, _ := role.Key.code()
	flag := nations.Flag(code)
	return b.names.NameOf(flag), cpflags.IsOpenedCode(code)
}

func (b *Book) IDOf(ctx context.Context, role Role) (discord.RoleID, error) {
	kept, err := b.store.RoleOf(ctx, role.Key)
	if err != nil {
		return "", fmt.Errorf("failed to read the role %s: %w", role.Key, err)
	}
	return kept.ID, nil
}

// Makes the role, or paints the one kept when its color is not the role's and renames a sold flag's when its name is
// not the flag's. A role someone deleted by hand in the server is made again.
func (b *Book) Ensure(ctx context.Context, role Role) (discord.RoleID, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	kept, err := b.store.RoleOf(ctx, role.Key)
	if errors.Is(err, ErrNoRole) {
		return b.make(ctx, role)
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the role %s: %w", role.Key, err)
	}
	return b.restyle(ctx, role, kept)
}

// Refresh renames and paints the role kept, and makes none: a flag nobody linked needs no role.
func (b *Book) Refresh(ctx context.Context, role Role) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	kept, err := b.store.RoleOf(ctx, role.Key)
	if errors.Is(err, ErrNoRole) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read the role %s: %w", role.Key, err)
	}
	_, err = b.restyle(ctx, role, kept)
	return err
}

func (b *Book) restyle(ctx context.Context, role Role, kept Kept) (discord.RoleID, error) {
	name, follows := b.nameOf(role)
	renamed := follows && kept.Name != name
	if kept.Color == role.Color && !renamed {
		return kept.ID, nil
	}

	var err error
	if kept.Color != role.Color {
		err = b.maker.PaintRole(ctx, kept.ID, role.Color)
	}
	if err == nil && renamed {
		err = b.maker.RenameRole(ctx, kept.ID, name)
	}
	if errors.Is(err, discord.ErrNoRole) {
		return b.make(ctx, role)
	}
	if err != nil {
		return "", fmt.Errorf("failed to restyle the role %s: %w", role.Key, err)
	}
	if follows {
		kept.Name = name
	}
	kept.Color = role.Color
	if err := b.store.SaveRole(ctx, role.Key, kept); err != nil {
		return "", fmt.Errorf("failed to keep the role %s: %w", role.Key, err)
	}
	return kept.ID, nil
}

func (b *Book) make(ctx context.Context, role Role) (discord.RoleID, error) {
	name, follows := b.nameOf(role)
	id, err := b.maker.MakeRole(ctx, name, role.Color)
	if err != nil {
		return "", fmt.Errorf("failed to make the role %s: %w", role.Key, err)
	}
	kept := Kept{ID: id, Color: role.Color}
	if follows {
		kept.Name = name
	}
	if err := b.store.SaveRole(ctx, role.Key, kept); err != nil {
		return "", fmt.Errorf("failed to keep the role %s: %w", role.Key, err)
	}
	return id, nil
}

// Every role the bot made, as the game names it now: a flag's in its color (a sold flag's under its name), and any
// other deleted from the server.
func (b *Book) Tidy(ctx context.Context) error {
	keys, err := b.store.Keys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list the roles: %w", err)
	}
	var errs []error
	for _, key := range keys {
		code, ok := key.code()
		if flag, known := b.names.FlagOf(code); ok && known {
			if _, err := b.Ensure(ctx, Nation(flag)); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := b.retire(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (b *Book) retire(ctx context.Context, key Key) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	kept, err := b.store.RoleOf(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to read the role %s: %w", key, err)
	}
	if err := b.maker.DeleteRole(ctx, kept.ID); err != nil {
		return fmt.Errorf("failed to delete the role %s: %w", key, err)
	}
	if err := b.store.ForgetRole(ctx, key); err != nil {
		return fmt.Errorf("failed to forget the role %s: %w", key, err)
	}
	return nil
}
