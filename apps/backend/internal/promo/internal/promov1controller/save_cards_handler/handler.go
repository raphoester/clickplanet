package save_cards_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/backgrounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
)

type UseCase interface {
	Execute(ctx context.Context, set cards.SetName, drafts []cards.Draft) error
}

func New(useCase UseCase) SaveCardsHandler {
	return SaveCardsHandler{useCase: useCase}
}

type SaveCardsHandler struct {
	useCase UseCase
}

func (h SaveCardsHandler) SaveCards(
	ctx context.Context,
	req *connect.Request[promov1.SaveCardsRequest],
) (*connect.Response[promov1.SaveCardsResponse], error) {
	drafts := make([]cards.Draft, 0, len(req.Msg.GetCards()))
	for _, card := range req.Msg.GetCards() {
		id, err := backgrounds.BackgroundIDOf(card.GetBackgroundId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		drafts = append(drafts, cards.Draft{Background: id, Headline: card.GetHeadline(), Line: card.GetLine()})
	}
	err := h.useCase.Execute(ctx, cards.SetName(req.Msg.GetRuleSet()), drafts)
	switch {
	case errors.Is(err, cards.ErrInvalidCards):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, cards.ErrUnknownSet), errors.Is(err, backgrounds.ErrNoBackground):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&promov1.SaveCardsResponse{}), nil
}
