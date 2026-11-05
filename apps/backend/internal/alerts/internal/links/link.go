package links

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

var ErrInvalidAccount = errors.New("not an account id")

func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}

var ErrNoLink = errors.New("this account has no Discord link")

// An account signed in with Discord, and its Discord user: in the server or not, linked or not.
type DiscordAccount struct {
	Account AccountID
	User    discord.UserID
}

// An account's username, as player shows it to the other players: the nickname the bot gives its Discord user in the
// server. Empty: none to show, for a guest or an account its player asked to delete.
type Name string

// A player's Discord user, in the server with the role of the flag it pledged and its username as its nickname. No
// flag yet: the account never pledged one, and the first pledge gives it. No name: the bot gave the user none yet.
type Link struct {
	account   AccountID
	user      discord.UserID
	flag      nations.Flag
	name      Name
	linkedAt  time.Time
	forgotten bool
}

func NewLink(account AccountID, user discord.UserID, flag nations.Flag, now time.Time) Link {
	return Link{account: account, user: user, flag: flag, linkedAt: now}
}

// The roles the user's flag changes: no From when it had none, no To when it holds none after.
type Move struct {
	User discord.UserID
	From nations.Flag
	To   nations.Flag
}

func (l *Link) Follow(flag nations.Flag) (Move, bool) {
	if flag == l.flag {
		return Move{}, false
	}
	move := Move{User: l.user, From: l.flag, To: flag}
	l.flag = flag
	return move, true
}

// What the user is called in the server: no name takes the nickname off, and Discord shows its own name again.
type Nickname struct {
	User discord.UserID
	Name Name
}

// The user wears the account's username as its nickname, and none for no name: false when it wears it already.
func (l *Link) Rename(name Name) (Nickname, bool) {
	if name == l.name {
		return Nickname{}, false
	}
	l.name = name
	return Nickname{User: l.user, Name: name}, true
}

// The flag's role leaves the user with the link.
func (l *Link) Forget() Move {
	l.forgotten = true
	return Move{User: l.user, From: l.flag}
}

func (l Link) BelongsTo(account AccountID) bool {
	return l.account == account
}

// The same account with the same Discord user: a join again changes nothing.
func (l Link) Joins(account AccountID, user discord.UserID) bool {
	return l.account == account && l.user == user
}

type LinkSnapshot struct {
	Account   AccountID
	User      discord.UserID
	Flag      nations.Flag
	Name      Name
	LinkedAt  time.Time
	Forgotten bool
}

func (l Link) TakeSnapshot() LinkSnapshot {
	return LinkSnapshot{
		Account: l.account, User: l.user, Flag: l.flag, Name: l.name, LinkedAt: l.linkedAt, Forgotten: l.forgotten,
	}
}

func (s LinkSnapshot) Restore() Link {
	return Link{
		account: s.Account, user: s.User, flag: s.Flag, name: s.Name, linkedAt: s.LinkedAt, forgotten: s.Forgotten,
	}
}

// Who an account is to the other players, from player.
type Names interface {
	NameOf(ctx context.Context, account AccountID) (Name, error)
}

type Store interface {
	LinkOf(ctx context.Context, account AccountID) (Link, error)
	LinkOfUser(ctx context.Context, user discord.UserID) (Link, error)
	// A link kept with the same user keeps its name: SaveName alone changes it.
	SaveLinks(ctx context.Context, links ...Link) error
	// The link's name alone, so a nickname given while the flag moves keeps the flag's move: nothing when the account
	// is no longer linked to that user.
	SaveName(ctx context.Context, link Link) error
}
