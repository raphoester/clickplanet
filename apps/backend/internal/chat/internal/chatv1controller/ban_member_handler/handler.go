// Package ban_member_handler serves chat.v1.AdminService/BanMember.
package ban_member_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/ban_member_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatban"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type UseCase interface {
	Execute(ctx context.Context, in ban_member_usecase.In) (ban_member_usecase.Out, error)
}

func New(useCase UseCase) BanMemberHandler {
	return BanMemberHandler{useCase: useCase}
}

type BanMemberHandler struct {
	useCase UseCase
}

func (h BanMemberHandler) BanMember(
	ctx context.Context,
	req *connect.Request[chatv1.BanMemberRequest],
) (*connect.Response[chatv1.BanMemberResponse], error) {
	out, err := h.useCase.Execute(ctx, ban_member_usecase.In{
		AuthorTag: req.Msg.GetAuthorTag(),
		Reason:    req.Msg.GetReason(),
	})
	if err != nil {
		// An operator is told what they got wrong; only a sender is kept guessing.
		if errors.Is(err, messages.ErrInvalidTag) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, err
	}

	return connect.NewResponse(&chatv1.BanMemberResponse{
		Ban:      chatban.Encode(out.Ban),
		Redacted: uint32(out.Redacted), //nolint:gosec // a history of at most a few hundred messages.
	}), nil
}
