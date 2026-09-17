// Package rpc_player_usernames asks the player module for the username an account chose, over the internal
// listener.
package rpc_player_usernames

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// Dialer is cpbootstrap's internal listener: the only way one module reaches another.
type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

// askTimeout bounds one call, which a post waits on. It is loopback, so this is a stuck socket rather than a
// slow network, and the post goes out as a guest's.
const askTimeout = time.Second

type Usernames struct {
	dial Dialer
}

func New(dial Dialer) *Usernames {
	return &Usernames{dial: dial}
}

// Username asks on every post, and keeps nothing: a player may choose a name at any time, and posts are few.
func (u *Usernames) Username(ctx context.Context, account messages.AccountID) (string, bool, error) {
	client, baseURL, err := u.dial.Dial()
	if err != nil {
		return "", false, fmt.Errorf("failed to reach the player module: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := playerv1connect.NewInternalServiceClient(client, baseURL).
		GetNames(ctx, connect.NewRequest(&playerv1.GetNamesRequest{AccountIds: []string{account.String()}}))
	if err != nil {
		return "", false, fmt.Errorf("failed to ask the player module for the username: %w", err)
	}

	username, found := res.Msg.GetNames()[account.String()]
	return username, found && username != "", nil
}
