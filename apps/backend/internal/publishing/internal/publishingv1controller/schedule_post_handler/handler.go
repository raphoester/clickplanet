package schedule_post_handler

import (
	"context"
	"errors"
	"slices"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts/usecases/schedule_post_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/publishingv1controller/publishingwire"
)

type UseCase interface {
	Execute(ctx context.Context, post posts.Post) schedule_post_usecase.Outcome
}

func New(useCase UseCase) SchedulePostHandler {
	return SchedulePostHandler{useCase: useCase}
}

type SchedulePostHandler struct {
	useCase UseCase
}

func (h SchedulePostHandler) SchedulePost(
	ctx context.Context,
	req *connect.Request[publishingv1.SchedulePostRequest],
) (*connect.Response[publishingv1.SchedulePostResponse], error) {
	post, err := postOf(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(answerOf(h.useCase.Execute(ctx, post))), nil
}

func postOf(msg *publishingv1.SchedulePostRequest) (posts.Post, error) {
	if msg.GetSlot() == nil {
		return posts.Post{}, errors.New("no slot")
	}
	platforms, err := publishingwire.Platforms(msg.GetSlot().GetPlatforms())
	if err != nil {
		return posts.Post{}, err //nolint:wrapcheck // the wire already says which platform is wrong.
	}
	draft := posts.Draft{
		Slot:  calendar.Slot{Platforms: platforms},
		Media: posts.MediaURL(msg.GetMediaUrl()), Key: posts.IdempotencyKey(msg.GetIdempotencyKey()), Title: msg.GetTitle(),
	}
	// Absent is no time, which the post refuses, and not 1970.
	if msg.GetSlot().GetAt() != nil {
		draft.Slot.At = msg.GetSlot().GetAt().AsTime()
	}
	if msg.GetMetadata() != nil {
		draft.Metadata = msg.GetMetadata().AsMap()
	}
	if msg.GetTiktok() != nil {
		draft.TikTok = &posts.TikTokWords{Caption: msg.GetTiktok().GetCaption()}
	}
	if msg.GetYoutube() != nil {
		draft.YouTube = &posts.YouTubeWords{
			Title: msg.GetYoutube().GetTitle(), Description: msg.GetYoutube().GetDescription(),
			Tags: slices.Clone(msg.GetYoutube().GetTags()),
		}
	}
	return posts.NewPost(draft) //nolint:wrapcheck // the domain says every reason.
}

func answerOf(outcome schedule_post_usecase.Outcome) *publishingv1.SchedulePostResponse {
	switch {
	case outcome.Scheduled != nil:
		scheduled := &publishingv1.Scheduled{PostId: string(outcome.Scheduled.ID)}
		for _, platform := range []calendar.Platform{calendar.TikTok, calendar.YouTube} {
			if why, refused := outcome.Scheduled.Refused[platform]; refused {
				scheduled.Refusals = append(scheduled.Refusals, &publishingv1.Refusal{
					Platform: publishingwire.Platform(platform), Reason: why.Error(),
				})
			}
		}
		return &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Scheduled{Scheduled: scheduled}}
	case outcome.Busy != nil:
		busy := &publishingv1.Busy{Reason: outcome.Busy.Reason}
		if outcome.Busy.RetryAfter > 0 {
			busy.RetryAfter = durationpb.New(outcome.Busy.RetryAfter)
		}
		return &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Busy{Busy: busy}}
	case outcome.Taken != "":
		return &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Taken{
			Taken: &publishingv1.Taken{Reason: outcome.Taken},
		}}
	default:
		return &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Failed{
			Failed: &publishingv1.Failed{Reason: outcome.Failed},
		}}
	}
}
