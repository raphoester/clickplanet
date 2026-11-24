package list_letters_handler

import (
	"context"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_letters_handler/letters_query"
)

type Query interface {
	Letters(ctx context.Context, filter letters_query.Filter) (*supportv1.ListLettersResponse, error)
}

func New(query Query) ListLettersHandler {
	return ListLettersHandler{query: query}
}

type ListLettersHandler struct {
	query Query
}

const (
	defaultLimit = 100
	maxLimit     = 500
)

func (h ListLettersHandler) ListLetters(
	ctx context.Context,
	req *connect.Request[supportv1.ListLettersRequest],
) (*connect.Response[supportv1.ListLettersResponse], error) {
	filter := letters_query.Filter{OpenOnly: req.Msg.GetOpenOnly(), Limit: defaultLimit}
	if limit := req.Msg.GetLimit(); limit > 0 {
		filter.Limit = min(int(limit), maxLimit)
	}
	if account := req.Msg.GetAccountId(); account != "" {
		id, err := notes.AccountIDOf(account)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, notes.ErrInvalidAccount)
		}
		filter.Account = &id
	}
	if submission := req.Msg.GetSubmissionId(); submission != "" {
		id, err := letters.SubmissionIDOf(submission)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, letters.ErrInvalidSubmission)
		}
		filter.Submission = &id
	}
	found, err := h.query.Letters(ctx, filter)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(found), nil
}
