package list_messages_handler

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_messages_handler/messages_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// Read as everyone, it would answer a page of other people's messages to an operator reading one account's.
var ErrInvalidAccount = errors.New("the account id is not a UUID")

type Query interface {
	Messages(
		ctx context.Context, cursor messages_query.Cursor, author messages.AccountID, limit uint32,
	) (*chatv1.ListMessagesResponse, error)
}

func New(query Query) ListMessagesHandler {
	return ListMessagesHandler{query: query}
}

type ListMessagesHandler struct {
	query Query
}

func (h ListMessagesHandler) ListMessages(
	ctx context.Context,
	req *connect.Request[chatv1.ListMessagesRequest],
) (*connect.Response[chatv1.ListMessagesResponse], error) {
	author := messages.AccountIDOf(req.Msg.GetAccountId())
	if req.Msg.GetAccountId() != "" && author == messages.NoAccount {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrInvalidAccount)
	}

	cursor := messages_query.Cursor{Before: req.Msg.GetBeforeMessageId(), After: req.Msg.GetAfterMessageId()}
	page, err := h.query.Messages(ctx, cursor, author, req.Msg.GetLimit())
	if errors.Is(err, messages_query.ErrTwoCursors) {
		return nil, connect.NewError(connect.CodeInvalidArgument, messages_query.ErrTwoCursors)
	}
	if errors.Is(err, messages_query.ErrNoMessage) {
		return nil, connect.NewError(connect.CodeNotFound, messages_query.ErrNoMessage)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read a page of messages: %w", err)
	}

	res := connect.NewResponse(page)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
