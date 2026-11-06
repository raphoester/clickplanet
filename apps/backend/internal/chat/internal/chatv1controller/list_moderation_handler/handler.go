package list_moderation_handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_moderation_handler/moderation_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// Read as everyone, it would answer other people's actions to an operator reading one account's.
var ErrInvalidAccount = errors.New("the account id is not a UUID")

type Query interface {
	Actions(ctx context.Context, filter moderation_query.Filter) (*chatv1.ListModerationResponse, error)
}

func New(query Query) ListModerationHandler {
	return ListModerationHandler{query: query}
}

type ListModerationHandler struct {
	query Query
}

func (h ListModerationHandler) ListModeration(
	ctx context.Context,
	req *connect.Request[chatv1.ListModerationRequest],
) (*connect.Response[chatv1.ListModerationResponse], error) {
	account := messages.AccountIDOf(req.Msg.GetAccountId())
	if req.Msg.GetAccountId() != "" && account == messages.NoAccount {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidAccount)
	}

	filter := moderation_query.Filter{Account: account, Limit: req.Msg.GetLimit()}
	if since := req.Msg.GetSince(); since != nil {
		filter.Since = since.AsTime()
	} else {
		filter.Since = time.Unix(0, 0)
	}

	history, err := h.query.Actions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to read the moderation history: %w", err)
	}

	res := connect.NewResponse(history)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
