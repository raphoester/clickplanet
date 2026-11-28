package rpc_planet_allegiance

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

type Planet interface {
	GetAllegianceOf(
		ctx context.Context,
		req *connect.Request[planetv1.GetAllegianceOfRequest],
	) (*connect.Response[planetv1.GetAllegianceOfResponse], error)
}

const askTimeout = 5 * time.Second

// nations.Catalog.
type Flags interface {
	FlagOf(code string) (nations.Flag, bool)
}

func New(planet Planet, flags Flags) Allegiances {
	return Allegiances{planet: planet, flags: flags}
}

type Allegiances struct {
	planet Planet
	flags  Flags
}

// No flag when the account never pledged one, or plays for one with no role.
func (a Allegiances) FlagOf(ctx context.Context, account links.AccountID) (nations.Flag, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := a.planet.GetAllegianceOf(ctx, connect.NewRequest(&planetv1.GetAllegianceOfRequest{AccountId: account.String()}))
	if err != nil {
		return "", fmt.Errorf("failed to ask planet the flag %s plays for: %w", account, err)
	}
	pledged := res.Msg.GetAllegiance().GetCountryId()
	if pledged == "" {
		return "", nil
	}
	flag, _ := a.flags.FlagOf(pledged)
	return flag, nil
}
