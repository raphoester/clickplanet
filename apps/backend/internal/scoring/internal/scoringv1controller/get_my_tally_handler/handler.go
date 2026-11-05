package get_my_tally_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_my_tally_handler/my_tally_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
)

type Query interface {
	MyTally(ctx context.Context, book books.BookID, account standings.AccountID, country string) (*scoringv1.GetMyTallyResponse, error)
}

func New(query Query) GetMyTallyHandler {
	return GetMyTallyHandler{query: query}
}

type GetMyTallyHandler struct {
	query Query
}

func (h GetMyTallyHandler) GetMyTally(
	ctx context.Context,
	req *connect.Request[scoringv1.GetMyTallyRequest],
) (*connect.Response[scoringv1.GetMyTallyResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}
	book, err := books.ParseBookID(req.Msg.GetBookId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	mine, err := h.query.MyTally(ctx, book, account, req.Msg.GetCountryId())
	if errors.Is(err, my_tally_query.ErrUnknownCountry) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}
	return connect.NewResponse(mine), nil
}
