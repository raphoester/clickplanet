// Package claim_bonus_handler serves planet.v1.ClickService/ClaimBonus.
package claim_bonus_handler

import (
	"context"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/clickbudget"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/claim_bonus"
)

type UseCase interface {
	Execute(ctx context.Context, in claim_bonus.In) (claim_bonus.Out, error)
}

// New takes a nil use case for a server with boxes switched off: ClaimBonus
// then answers Unimplemented, so the capability is absent rather than present
// and refusing.
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
	if h.useCase == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, claim_bonus.ErrNoSuchBonus)
	}

	out, err := h.useCase.Execute(ctx, claim_bonus.In{
		Token:     req.Msg.GetToken(),
		CountryID: req.Msg.GetCountryId(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, claim_bonus.ErrNoSuchBonus)
	}

	return connect.NewResponse(&planetv1.ClaimBonusResponse{
		Budget:          clickbudget.Encode(out.Budget),
		Kind:            EncodeKind(out.Kind),
		DurationSeconds: uint32(out.Duration / time.Second),
	}), nil
}

func EncodeKind(kind bonus.Kind) planetv1.BonusKind {
	switch kind {
	case bonus.KindTripleClicks:
		return planetv1.BonusKind_BONUS_KIND_TRIPLE_CLICKS
	case bonus.KindSpreadClicks:
		return planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS
	}

	return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED
}
