// Package list_bans_handler serves chat.v1.AdminService/ListBans.
package list_bans_handler

import (
	"context"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatban"
)

type UseCase interface {
	Execute(ctx context.Context) []bans.Ban
}

func New(useCase UseCase) ListBansHandler {
	return ListBansHandler{useCase: useCase}
}

type ListBansHandler struct {
	useCase UseCase
}

func (h ListBansHandler) ListBans(
	ctx context.Context,
	_ *connect.Request[chatv1.ListBansRequest],
) (*connect.Response[chatv1.ListBansResponse], error) {
	running := h.useCase.Execute(ctx)

	response := &chatv1.ListBansResponse{Bans: make([]*chatv1.Ban, 0, len(running))}
	for _, ban := range running {
		response.Bans = append(response.Bans, chatban.Encode(ban))
	}

	return connect.NewResponse(response), nil
}
