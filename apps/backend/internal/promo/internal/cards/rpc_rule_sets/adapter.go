package rpc_rule_sets

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
)

type Planet interface {
	GetRuleSetNames(context.Context, *connect.Request[planetv1.GetRuleSetNamesRequest]) (*connect.Response[planetv1.GetRuleSetNamesResponse], error)
	GetRuleSet(context.Context, *connect.Request[planetv1.GetRuleSetRequest]) (*connect.Response[planetv1.GetRuleSetResponse], error)
}

type Bonuses interface {
	GetRuleSet(context.Context, *connect.Request[bonusesv1.GetRuleSetRequest]) (*connect.Response[bonusesv1.GetRuleSetResponse], error)
}

func New(planet Planet, bonuses Bonuses) *Adapter {
	return &Adapter{planet: planet, bonuses: bonuses}
}

// The sets planet keeps, and the values each plays, in planet's words and the offers'.
type Adapter struct {
	planet  Planet
	bonuses Bonuses
}

// Names are the sets planet keeps: a card is a battle's, and a battle plays a set every module keeps.
func (a *Adapter) Names(ctx context.Context) ([]string, error) {
	res, err := a.planet.GetRuleSetNames(ctx, connect.NewRequest(&planetv1.GetRuleSetNamesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet which rule sets it keeps: %w", err)
	}
	return res.Msg.GetNames(), nil
}

// Played is cards.ErrUnknownSet for a set planet does not keep. A set the offers do not keep plays none of their values.
func (a *Adapter) Played(ctx context.Context, set string) (placeholders.Played, error) {
	res, err := a.planet.GetRuleSet(ctx, connect.NewRequest(&planetv1.GetRuleSetRequest{Name: set}))
	switch {
	case connect.CodeOf(err) == connect.CodeNotFound:
		return placeholders.Played{}, fmt.Errorf("%w: %q", cards.ErrUnknownSet, set)
	case err != nil:
		return placeholders.Played{}, fmt.Errorf("failed to ask planet what %q plays: %w", set, err)
	}
	if res.Msg.GetPlayed() == nil {
		return placeholders.Played{}, errors.New("planet answered no values")
	}
	played := placeholders.Played{Planet: res.Msg.GetPlayed()}

	offered, err := a.bonuses.GetRuleSet(ctx, connect.NewRequest(&bonusesv1.GetRuleSetRequest{Name: set}))
	switch {
	case connect.CodeOf(err) == connect.CodeNotFound:
		return played, nil
	case err != nil:
		return placeholders.Played{}, fmt.Errorf("failed to ask the bonuses module what %q plays: %w", set, err)
	}
	played.Offers = offered.Msg.GetPlayed()
	return played, nil
}
