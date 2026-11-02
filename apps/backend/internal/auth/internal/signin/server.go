package signin

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

// The game's Discord server. A sign-in whose claim carries a token that may join it adds the player there, once the
// sign-in is saved. It never fails the sign-in: a player who could not join is signed in all the same.
type Server interface {
	Join(ctx context.Context, account accounts.AccountID, user string, token string)
}

// Nobody joins: no sign-in asks leave to.
type NoServer struct{}

func (NoServer) Join(context.Context, accounts.AccountID, string, string) {}
