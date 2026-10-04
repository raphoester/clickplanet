package listen_for_events_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/listen_for_titles_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, sink listen_for_events_usecase.Sink) error
}

type TitlesUseCase interface {
	Execute(ctx context.Context, account players.AccountID, sink listen_for_titles_usecase.Sink) error
}

func New(roster UseCase, titles TitlesUseCase) ListenForEventsHandler {
	return ListenForEventsHandler{roster: roster, titles: titles}
}

type ListenForEventsHandler struct {
	roster UseCase
	titles TitlesUseCase
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	_ *connect.Request[playerv1.ListenForEventsRequest],
	stream *connect.ServerStream[playerv1.PlayerEvent],
) error {
	sink := NewSink(stream)

	account, err := caller.AccountOf(ctx)
	if err != nil {
		return h.roster.Execute(ctx, sink)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	titlesDone := make(chan struct{})
	go func() {
		defer close(titlesDone)
		_ = h.titles.Execute(ctx, account, sink)
	}()

	err = h.roster.Execute(ctx, sink)
	cancel()
	<-titlesDone
	return err
}
