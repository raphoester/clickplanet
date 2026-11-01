package get_season_replay_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/get_season_replay_handler/season_replay_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type Query interface {
	Replay(ctx context.Context, part season_replay_query.Part) (*planetv1.MapHistory, error)
}

func New(query Query) GetSeasonReplayHandler {
	return GetSeasonReplayHandler{query: query}
}

type GetSeasonReplayHandler struct {
	query Query
}

func (h GetSeasonReplayHandler) GetSeasonReplay(
	ctx context.Context,
	req *connect.Request[gamemasterv1.GetSeasonReplayRequest],
) (*connect.Response[gamemasterv1.GetSeasonReplayResponse], error) {
	history, err := h.query.Replay(ctx, season_replay_query.Part{Season: season.Number(req.Msg.GetSeason()), Finale: req.Msg.GetFinale()})
	if errors.Is(err, season.ErrNotEnded) || errors.Is(err, season_replay_query.ErrNoFinale) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(&gamemasterv1.GetSeasonReplayResponse{History: history}), nil
}
