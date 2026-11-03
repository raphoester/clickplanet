package rpc_planet_flags

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Planet interface {
	GetAllegianceOf(
		ctx context.Context,
		req *connect.Request[planetv1.GetAllegianceOfRequest],
	) (*connect.Response[planetv1.GetAllegianceOfResponse], error)
}

const askTimeout = time.Second

func New(planet Planet) Flags {
	return Flags{planet: planet}
}

type Flags struct {
	planet Planet
}

var _ messages.Flags = Flags{}

func (f Flags) Flag(ctx context.Context, account messages.AccountID) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := f.planet.GetAllegianceOf(ctx, connect.NewRequest(&planetv1.GetAllegianceOfRequest{AccountId: account.String()}))
	if err != nil {
		return "", fmt.Errorf("failed to ask planet the flag %s plays for: %w", account, err)
	}
	return res.Msg.GetAllegiance().GetCountryId(), nil
}
