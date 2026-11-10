package set_name_verdict_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/name_verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
)

type UseCase interface {
	Execute(ctx context.Context, account verdicts.AccountID, shown bool, reason string) (verdicts.Verdict, error)
}

func New(useCase UseCase) SetNameVerdictHandler {
	return SetNameVerdictHandler{useCase: useCase}
}

type SetNameVerdictHandler struct {
	useCase UseCase
}

func (h SetNameVerdictHandler) SetNameVerdict(
	ctx context.Context,
	req *connect.Request[clipsv1.SetNameVerdictRequest],
) (*connect.Response[clipsv1.SetNameVerdictResponse], error) {
	account, err := casts.AccountIDOf(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	set, err := h.useCase.Execute(ctx, account, req.Msg.GetShown(), req.Msg.GetReason())
	switch {
	case errors.Is(err, verdicts.ErrNoVerdict):
		return nil, connect.NewError(connect.CodeNotFound, verdicts.ErrNoVerdict)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	verdict, err := name_verdicts.ProtoOf(set.TakeSnapshot())
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&clipsv1.SetNameVerdictResponse{Verdict: verdict}), nil
}
