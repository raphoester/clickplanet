package links

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

var ErrNoLink = errors.New("this account has no Discord link")

// A player's Discord user, in the server with the role of the flag it last took a tile for. No flag yet: the account
// never pledged one, and the first take gives it.
type Link struct {
	account   AccountID
	user      discord.UserID
	flag      nations.Flag
	linkedAt  time.Time
	forgotten bool
}

func NewLink(account AccountID, user discord.UserID, flag nations.Flag, now time.Time) Link {
	return Link{account: account, user: user, flag: flag, linkedAt: now}
}

// The roles the user's flag changes: no From when it had none.
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

func (l *Link) Forget() {
	l.forgotten = true
}

func (l Link) BelongsTo(account AccountID) bool {
	return l.account == account
}

type LinkSnapshot struct {
	Account   AccountID
	User      discord.UserID
	Flag      nations.Flag
	LinkedAt  time.Time
	Forgotten bool
}

func (l Link) TakeSnapshot() LinkSnapshot {
	return LinkSnapshot{Account: l.account, User: l.user, Flag: l.flag, LinkedAt: l.linkedAt, Forgotten: l.forgotten}
}

func (s LinkSnapshot) Restore() Link {
	return Link{account: s.Account, user: s.User, flag: s.Flag, linkedAt: s.LinkedAt, forgotten: s.Forgotten}
}

type Store interface {
	FlowOf(ctx context.Context, account AccountID) (Flow, error)
	SaveFlow(ctx context.Context, flow Flow) error
	LinkOf(ctx context.Context, account AccountID) (Link, error)
	LinkOfUser(ctx context.Context, user discord.UserID) (Link, error)
	SaveLinks(ctx context.Context, links ...Link) error
}
