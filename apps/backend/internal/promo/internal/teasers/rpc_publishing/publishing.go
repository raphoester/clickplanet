// The publishing module, as promo asks it: its InternalService, on the loopback internal listener.
package rpc_publishing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

func New(client publishingv1connect.InternalServiceClient) Publishing {
	return Publishing{client: client}
}

type Publishing struct {
	client publishingv1connect.InternalServiceClient
}

var _ teasers.Publishing = Publishing{}

// The battles the posts since then tease, whatever became of each post.
func (p Publishing) Teased(ctx context.Context, since time.Time) ([]teasers.BattleID, error) {
	answer, err := p.client.ListPosts(ctx, connect.NewRequest(&publishingv1.ListPostsRequest{Since: timestamppb.New(since)}))
	if err != nil {
		return nil, failure("list the posts", err)
	}
	var teased []teasers.BattleID
	for _, post := range answer.Msg.GetPosts() {
		battle := teasers.BattleID(post.GetMetadata().GetFields()[teasers.MetadataBattle].GetStringValue())
		if battle != "" && !slices.Contains(teased, battle) {
			teased = append(teased, battle)
		}
	}
	return teased, nil
}

// The first slot from notBefore to by at which every one of the platforms takes the post.
func (p Publishing) Slot(ctx context.Context, platforms []teasers.Platform, notBefore, by time.Time) (teasers.Slot, bool, error) {
	answer, err := p.client.PlanPost(ctx, connect.NewRequest(&publishingv1.PlanPostRequest{
		Platforms: wireOf(platforms), NotBefore: timestamppb.New(notBefore), By: timestamppb.New(by),
	}))
	if err != nil {
		return teasers.Slot{}, false, failure("plan a post", err)
	}
	if answer.Msg.GetSlot() == nil {
		return teasers.Slot{}, false, nil
	}
	slot := teasers.Slot{At: answer.Msg.GetSlot().GetAt().AsTime()}
	for _, platform := range answer.Msg.GetSlot().GetPlatforms() {
		slot.Platforms = append(slot.Platforms, platformOf(platform))
	}
	return slot, true, nil
}

// The post at its slot: the video for every platform, and the photos as TikTok's carousel when TikTok is in it.
func (p Publishing) Schedule(ctx context.Context, post teasers.Post) (teasers.Receipt, error) {
	keeps, err := structpb.NewStruct(post.Metadata())
	if err != nil {
		return teasers.Receipt{}, fmt.Errorf("failed to write what the post keeps: %w", err)
	}
	var photos []string
	if slices.Contains(post.Slot().Platforms, teasers.TikTok) {
		for _, photo := range post.Media().Photos {
			photos = append(photos, string(photo))
		}
	}
	words := post.Words()
	answer, err := p.client.SchedulePost(ctx, connect.NewRequest(&publishingv1.SchedulePostRequest{
		Slot:     &publishingv1.Slot{At: timestamppb.New(post.Slot().At), Platforms: wireOf(post.Slot().Platforms)},
		MediaUrl: string(post.Media().Video), IdempotencyKey: post.Key(), Title: words.Title,
		Tiktok: &publishingv1.TikTokWords{Caption: words.TikTok.Caption, Title: words.TikTok.Title},
		Youtube: &publishingv1.YouTubeWords{
			Title: words.YouTube.Title, Description: words.YouTube.Description, Tags: words.YouTube.Tags,
		},
		TiktokPhotoUrls: photos,
		Metadata:        keeps,
	}))
	if err != nil {
		return teasers.Receipt{}, failure("schedule the post", err)
	}
	return receiptOf(answer.Msg)
}

func receiptOf(answer *publishingv1.SchedulePostResponse) (teasers.Receipt, error) {
	switch outcome := answer.GetOutcome().(type) {
	case *publishingv1.SchedulePostResponse_Scheduled:
		receipt := teasers.Receipt{ID: outcome.Scheduled.GetPostId(), Refused: map[teasers.Platform]string{}}
		for _, refusal := range outcome.Scheduled.GetRefusals() {
			receipt.Refused[platformOf(refusal.GetPlatform())] = refusal.GetReason()
		}
		return receipt, nil
	case *publishingv1.SchedulePostResponse_Busy:
		return teasers.Receipt{}, &teasers.BusyError{RetryAfter: outcome.Busy.GetRetryAfter().AsDuration(), Reason: outcome.Busy.GetReason()}
	case *publishingv1.SchedulePostResponse_Taken:
		return teasers.Receipt{}, fmt.Errorf("%w: %s", teasers.ErrTaken, outcome.Taken.GetReason())
	case *publishingv1.SchedulePostResponse_Failed:
		return teasers.Receipt{}, &teasers.RefusedError{Reason: outcome.Failed.GetReason()}
	default:
		return teasers.Receipt{}, errors.New("publishing said nothing of the post")
	}
}

// Busy when publishing could not answer now; any other refusal as publishing said it, which the error net passes on.
func failure(what string, err error) error {
	switch connect.CodeOf(err) {
	case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeResourceExhausted, connect.CodeAborted:
		return &teasers.BusyError{Reason: fmt.Sprintf("publishing could not %s: %s", what, err)}
	default:
		return fmt.Errorf("publishing refused to %s: %w", what, err)
	}
}

func wireOf(platforms []teasers.Platform) []publishingv1.Platform {
	wire := make([]publishingv1.Platform, 0, len(platforms))
	for _, platform := range platforms {
		switch platform {
		case teasers.TikTok:
			wire = append(wire, publishingv1.Platform_PLATFORM_TIKTOK)
		case teasers.YouTube:
			wire = append(wire, publishingv1.Platform_PLATFORM_YOUTUBE)
		default:
			wire = append(wire, publishingv1.Platform_PLATFORM_UNSPECIFIED)
		}
	}
	return wire
}

func platformOf(platform publishingv1.Platform) teasers.Platform {
	switch platform {
	case publishingv1.Platform_PLATFORM_TIKTOK:
		return teasers.TikTok
	case publishingv1.Platform_PLATFORM_YOUTUBE:
		return teasers.YouTube
	default:
		return teasers.Platform(strings.ToLower(strings.TrimPrefix(platform.String(), "PLATFORM_")))
	}
}
