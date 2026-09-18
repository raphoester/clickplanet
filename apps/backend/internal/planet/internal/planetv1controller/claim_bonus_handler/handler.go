// Package claim_bonus_handler serves planet.v1.ClickService/ClaimBonus.
package claim_bonus_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/claim_bonus_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/clickbudget"
)

type UseCase interface {
	Execute(ctx context.Context, in claim_bonus_usecase.In) (claim_bonus_usecase.Out, error)
}

func New(useCase UseCase) ClaimBonusHandler {
	return ClaimBonusHandler{useCase: useCase}
}

type ClaimBonusHandler struct {
	useCase UseCase
}

func (h ClaimBonusHandler) ClaimBonus(
	ctx context.Context,
	req *connect.Request[planetv1.ClaimBonusRequest],
) (*connect.Response[planetv1.ClaimBonusResponse], error) {
	out, err := h.useCase.Execute(ctx, claim_bonus_usecase.In{
		Token:     req.Msg.GetToken(),
		CountryID: req.Msg.GetCountryId(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, claim_bonus_usecase.ErrNoSuchBonus)
	}

	response := &planetv1.ClaimBonusResponse{
		Budget:            clickbudget.Encode(out.Budget),
		Kind:              EncodeKind(out.Kind),
		DurationSeconds:   uint32(out.Duration / time.Second),
		EnclosureMaxTiles: uint32(out.EnclosureMaxTiles),
		BlastRadius:       out.BlastRadius,
		Charges:           chargesheld.Encode(out.Held),
	}
	if out.Kind == bonuses.KindEncloseClicks {
		// A charge is one shape. Said for a client that still counts shapes.
		response.Enclosures = 1
	}

	return connect.NewResponse(response), nil
}

func EncodeKind(kind bonuses.Kind) planetv1.BonusKind {
	switch kind {
	case bonuses.KindTripleClicks:
		return planetv1.BonusKind_BONUS_KIND_TRIPLE_CLICKS
	case bonuses.KindSpreadClicks:
		return planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS
	case bonuses.KindBomb:
		return planetv1.BonusKind_BONUS_KIND_BOMB
	case bonuses.KindEncloseClicks:
		return planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS
	}

	return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED
}
