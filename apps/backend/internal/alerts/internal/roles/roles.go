package roles

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

var ErrNoRole = errors.New("the bot made no such role")

type Key string

// A role the bot makes in the server the first time one is needed, kept by its key so a new name changes nothing.
type Role struct {
	Key  Key
	Name string
}

func Battles() Role {
	return Role{Key: "battles", Name: "Battles"}
}

func Nation(flag nations.Flag) Role {
	return Role{Key: Key("flag:" + string(flag)), Name: flag.Name()}
}

type Store interface {
	RoleOf(ctx context.Context, key Key) (discord.RoleID, error)
	SaveRole(ctx context.Context, key Key, id discord.RoleID) error
}

type Maker interface {
	MakeRole(ctx context.Context, name string) (discord.RoleID, error)
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
	id, err := b.store.RoleOf(ctx, role.Key)
	if err != nil {
		return "", fmt.Errorf("failed to read the role %s: %w", role.Key, err)
	}
	return id, nil
}

func (b *Book) Ensure(ctx context.Context, role Role) (discord.RoleID, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id, err := b.store.RoleOf(ctx, role.Key)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, ErrNoRole) {
		return "", fmt.Errorf("failed to read the role %s: %w", role.Key, err)
	}

	id, err = b.maker.MakeRole(ctx, role.Name)
	if err != nil {
		return "", fmt.Errorf("failed to make the role %s: %w", role.Key, err)
	}
	if err := b.store.SaveRole(ctx, role.Key, id); err != nil {
		return "", fmt.Errorf("failed to keep the role %s: %w", role.Key, err)
	}
	return id, nil
}
