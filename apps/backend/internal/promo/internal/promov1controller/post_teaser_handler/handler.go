package post_teaser_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/post_teaser_usecase"
)

func New(useCase post_teaser_usecase.Executor) PostTeaserHandler {
	return PostTeaserHandler{useCase: useCase}
}

type PostTeaserHandler struct {
	useCase post_teaser_usecase.Executor
}

var errNoBattle = errors.New("the request names no battle")

func (h PostTeaserHandler) PostTeaser(
	ctx context.Context,
	req *connect.Request[promov1.PostTeaserRequest],
) (*connect.Response[promov1.PostTeaserResponse], error) {
	if req.Msg.GetBattleId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoBattle)
	}
	media := teasers.Media{Video: teasers.MediaURL(req.Msg.GetVideoUrl())}
	for _, photo := range req.Msg.GetPhotoUrls() {
		media.Photos = append(media.Photos, teasers.MediaURL(photo))
	}

	outcome, err := h.useCase.Execute(ctx, teasers.BattleID(req.Msg.GetBattleId()), media)
	switch {
	case errors.Is(err, teasers.ErrInvalidPost):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers it, and passes a refusal of publishing's on.
	}
	return connect.NewResponse(answerOf(outcome)), nil
}

func answerOf(outcome teasers.Outcome) *promov1.PostTeaserResponse {
	not := &promov1.TeaserNotPosted{Reason: outcome.Why}
	switch outcome.Result {
	case teasers.Posted:
		posted := &promov1.TeaserPosted{PostId: outcome.Post, Title: outcome.Title, At: timestamppb.New(outcome.At)}
		for _, platform := range outcome.Platforms {
			posted.Platforms = append(posted.Platforms, string(platform))
			if why, refused := outcome.Refused[platform]; refused {
				posted.Refusals = append(posted.Refusals, &promov1.TeaserRefusal{Platform: string(platform), Reason: why})
			}
		}
		return &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Posted{Posted: posted}}
	case teasers.Held:
		return &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Held{Held: not}}
	case teasers.Skipped:
		return &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Skipped{Skipped: not}}
	case teasers.Busy:
		busy := &promov1.TeaserBusy{Reason: outcome.Why}
		if outcome.RetryAfter > 0 {
			busy.RetryAfter = durationpb.New(outcome.RetryAfter)
		}
		return &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Busy{Busy: busy}}
	default:
		return &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Failed{Failed: not}}
	}
}
