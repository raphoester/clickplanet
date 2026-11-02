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

func New(planet Planet) Allegiances {
	return Allegiances{planet: planet}
}

type Allegiances struct {
	planet Planet
}

// No flag when the account never pledged one.
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
	flag, err := nations.FlagOf(pledged)
	if err != nil {
		return "", fmt.Errorf("planet named a flag %s cannot play for: %w", account, err)
	}
	return flag, nil
}
