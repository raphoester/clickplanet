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
)

var ErrNoRole = errors.New("the bot made no such role")

type Key string

const flagPrefix = "flag:"

// A role the bot makes in the server the first time one is needed, kept by its key so a new name changes nothing.
type Role struct {
	Key   Key
	Name  string
	Color discord.Color
}

// The game names no other role: everybody in the server is pinged when a battle starts.
func Nation(flag nations.Flag) Role {
	return Role{Key: Key(flagPrefix + string(flag)), Name: flag.Name(), Color: ColorOf(flag)}
}

func (k Key) flag() (nations.Flag, bool) {
	code, found := strings.CutPrefix(string(k), flagPrefix)
	if !found {
		return "", false
	}
	flag, err := nations.FlagOf(code)
	return flag, err == nil
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

// What the store keeps of a role the bot made: its id, and the color it was last painted.
type Kept struct {
	ID    discord.RoleID
	Color discord.Color
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
	DeleteRole(ctx context.Context, role discord.RoleID) error
}

func NewBook(store Store, maker Maker) *Book {
	return &Book{store: store, maker: maker}
}

// One process makes each role once: two subscribers that need the same new role wait on each other.
type Book struct {
	mu    sync.Mutex
	store Store
	maker Maker
}

func (b *Book) IDOf(ctx context.Context, role Role) (discord.RoleID, error) {
	kept, err := b.store.RoleOf(ctx, role.Key)
	if err != nil {
		return "", fmt.Errorf("failed to read the role %s: %w", role.Key, err)
	}
	return kept.ID, nil
}

// Makes the role, or paints the one kept when its color is not the role's. A role someone deleted by hand in the
// server is made again.
func (b *Book) Ensure(ctx context.Context, role Role) (discord.RoleID, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	kept, err := b.store.RoleOf(ctx, role.Key)
	switch {
	case errors.Is(err, ErrNoRole):
		return b.make(ctx, role)
	case err != nil:
		return "", fmt.Errorf("failed to read the role %s: %w", role.Key, err)
	case kept.Color == role.Color:
		return kept.ID, nil
	}

	err = b.maker.PaintRole(ctx, kept.ID, role.Color)
	if errors.Is(err, discord.ErrNoRole) {
		return b.make(ctx, role)
	}
	if err != nil {
		return "", fmt.Errorf("failed to paint the role %s: %w", role.Key, err)
	}
	if err := b.store.SaveRole(ctx, role.Key, Kept{ID: kept.ID, Color: role.Color}); err != nil {
		return "", fmt.Errorf("failed to keep the role %s: %w", role.Key, err)
	}
	return kept.ID, nil
}

func (b *Book) make(ctx context.Context, role Role) (discord.RoleID, error) {
	id, err := b.maker.MakeRole(ctx, role.Name, role.Color)
	if err != nil {
		return "", fmt.Errorf("failed to make the role %s: %w", role.Key, err)
	}
	if err := b.store.SaveRole(ctx, role.Key, Kept{ID: id, Color: role.Color}); err != nil {
		return "", fmt.Errorf("failed to keep the role %s: %w", role.Key, err)
	}
	return id, nil
}

// Every role the bot made, as the game names it now: a flag's in its color, and any other deleted from the server.
func (b *Book) Tidy(ctx context.Context) error {
	keys, err := b.store.Keys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list the roles: %w", err)
	}
	var errs []error
	for _, key := range keys {
		if flag, ok := key.flag(); ok {
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
