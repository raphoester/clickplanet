package links

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

// Auth's id for the leave a Discord sign-in gave to join the server, as auth.v1.DiscordJoinGranted names it.
type GrantID string

var ErrNoGrant = errors.New("auth keeps no such grant: never made, or expired")

// What a Discord sign-in gave, which auth keeps for a few minutes: the account it signed in to, the Discord user, and
// a token that may join the server.
type Grant struct {
	Account AccountID
	User    discord.UserID
	Token   discord.AccessToken
}
