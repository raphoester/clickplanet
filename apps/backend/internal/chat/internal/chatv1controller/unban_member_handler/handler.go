// Package unban_member_handler serves chat.v1.AdminService/UnbanMember.
package unban_member_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/unban_member_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type UseCase interface {
	Execute(ctx context.Context, in unban_member_usecase.In) (unban_member_usecase.Out, error)
}

func New(useCase UseCase) UnbanMemberHandler {
	return UnbanMemberHandler{useCase: useCase}
}

type UnbanMemberHandler struct {
	useCase UseCase
}

func (h UnbanMemberHandler) UnbanMember(
	ctx context.Context,
	req *connect.Request[chatv1.UnbanMemberRequest],
) (*connect.Response[chatv1.UnbanMemberResponse], error) {
	out, err := h.useCase.Execute(ctx, unban_member_usecase.In{AuthorTag: req.Msg.GetAuthorTag()})
	if err != nil {
		return nil, toConnect(err)
	}

	return connect.NewResponse(&chatv1.UnbanMemberResponse{AuthorTag: out.AuthorTag}), nil
}

// toConnect tells an operator which of the two things they got wrong: a tag that
// is not one, or a member nobody had banned.
func toConnect(err error) error {
	switch {
	case errors.Is(err, messages.ErrInvalidTag):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, bans.ErrNotBanned):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return err
	}
}
