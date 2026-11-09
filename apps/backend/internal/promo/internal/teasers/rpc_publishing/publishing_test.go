package rpc_publishing_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1/publishingv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/rpc_publishing"
)

var (
	start = time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)
	at    = time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	both  = []teasers.Platform{teasers.TikTok, teasers.YouTube}
	wired = []publishingv1.Platform{publishingv1.Platform_PLATFORM_TIKTOK, publishingv1.Platform_PLATFORM_YOUTUBE}
)

// Publishing's internal service, answering as told and keeping what it was asked.
type fake struct {
	publishingv1connect.UnimplementedInternalServiceHandler

	err       error
	posts     []*publishingv1.Post
	slot      *publishingv1.Slot
	scheduled *publishingv1.SchedulePostResponse
	planned   *publishingv1.PlanPostRequest
	posted    *publishingv1.SchedulePostRequest
	since     time.Time
}

func (f *fake) ListPosts(_ context.Context, req *connect.Request[publishingv1.ListPostsRequest]) (*connect.Response[publishingv1.ListPostsResponse], error) {
	f.since = req.Msg.GetSince().AsTime()
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&publishingv1.ListPostsResponse{Posts: f.posts}), nil
}

func (f *fake) PlanPost(_ context.Context, req *connect.Request[publishingv1.PlanPostRequest]) (*connect.Response[publishingv1.PlanPostResponse], error) {
	f.planned = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&publishingv1.PlanPostResponse{Slot: f.slot}), nil
}

func (f *fake) SchedulePost(_ context.Context, req *connect.Request[publishingv1.SchedulePostRequest]) (*connect.Response[publishingv1.SchedulePostResponse], error) {
	f.posted = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.scheduled), nil
}

func serve(t *testing.T, f *fake) rpc_publishing.Publishing {
	t.Helper()
	_, handler := publishingv1connect.NewInternalServiceHandler(f)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return rpc_publishing.New(publishingv1connect.NewInternalServiceClient(server.Client(), server.URL))
}

func metadata(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()
	made, err := structpb.NewStruct(fields)
	require.NoError(t, err)
	return made
}

func ringWar(t *testing.T) teasers.Post {
	t.Helper()
	post, err := teasers.NewPost(teasers.Teaser{
		Battle: "entry-7", StartsAt: start, Cards: []teasers.Card{{Headline: "Circle any land"}, {Headline: "Gain the most land"}},
	}, teasers.Media{
		Photos: []teasers.MediaURL{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"},
		Video:  "https://media.example/teaser.mp4",
	})
	require.NoError(t, err)
	return post
}

func TestTheBattlesTeasedAreThoseThePostsMetadataNames(t *testing.T) {
	f := &fake{posts: []*publishingv1.Post{
		{PostId: "p-1", Status: "scheduled", Metadata: metadata(t, map[string]any{"teaserBattle": "entry-7"})},
		{PostId: "p-2", Status: "published", Metadata: metadata(t, map[string]any{"storyFlag": "fr"})},
		{PostId: "p-3", Status: "published"},
		{PostId: "p-4", Status: "cancelled", Metadata: metadata(t, map[string]any{"teaserBattle": "season-1/finale"})},
		{PostId: "p-5", Status: "failed", Metadata: metadata(t, map[string]any{"teaserBattle": "entry-7"})},
	}}
	since := time.Date(2026, 11, 11, 7, 43, 0, 0, time.UTC)

	teased, err := serve(t, f).Teased(t.Context(), since)

	require.NoError(t, err)
	assert.Equal(t, []teasers.BattleID{"entry-7", "season-1/finale"}, teased)
	assert.Equal(t, since, f.since)
}

func TestASlotIsAskedWithByAndNoneIsNotFound(t *testing.T) {
	f := &fake{slot: &publishingv1.Slot{At: timestamppb.New(at), Platforms: wired}}
	publishing := serve(t, f)

	slot, found, err := publishing.Slot(t.Context(), both, start.Add(-30*time.Hour), start.Add(-time.Hour))
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, teasers.Slot{At: at, Platforms: both}, slot)
	assert.Equal(t, wired, f.planned.GetPlatforms())
	assert.Equal(t, start.Add(-30*time.Hour), f.planned.GetNotBefore().AsTime())
	assert.Equal(t, start.Add(-time.Hour), f.planned.GetBy().AsTime())

	f.slot = nil
	_, found, err = publishing.Slot(t.Context(), both, at, start)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestAPostOnTikTokGoesWithItsPhotosItsWordsAndItsMetadata(t *testing.T) {
	f := &fake{scheduled: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Scheduled{
		Scheduled: &publishingv1.Scheduled{PostId: "post-1", Refusals: []*publishingv1.Refusal{
			{Platform: publishingv1.Platform_PLATFORM_YOUTUBE, Reason: "no YouTube account is connected"},
		}},
	}}}
	post := ringWar(t).At(teasers.Slot{At: at, Platforms: both})

	receipt, err := serve(t, f).Schedule(t.Context(), post)

	require.NoError(t, err)
	assert.Equal(t, teasers.Receipt{ID: "post-1", Refused: map[teasers.Platform]string{teasers.YouTube: "no YouTube account is connected"}}, receipt)
	words := post.Words()
	assert.Equal(t, at, f.posted.GetSlot().GetAt().AsTime())
	assert.Equal(t, wired, f.posted.GetSlot().GetPlatforms())
	assert.Equal(t, "https://media.example/teaser.mp4", f.posted.GetMediaUrl())
	assert.Equal(t, post.Key(), f.posted.GetIdempotencyKey())
	assert.Equal(t, words.Title, f.posted.GetTitle())
	assert.Equal(t, words.TikTok.Title, f.posted.GetTiktok().GetTitle())
	assert.Equal(t, words.TikTok.Caption, f.posted.GetTiktok().GetCaption())
	assert.Equal(t, words.YouTube.Title, f.posted.GetYoutube().GetTitle())
	assert.Equal(t, words.YouTube.Description, f.posted.GetYoutube().GetDescription())
	assert.Equal(t, words.YouTube.Tags, f.posted.GetYoutube().GetTags())
	assert.Equal(t, []string{"https://media.example/card-1.jpg", "https://media.example/card-2.jpg"}, f.posted.GetTiktokPhotoUrls())
	assert.Equal(t, "entry-7", f.posted.GetMetadata().GetFields()["teaserBattle"].GetStringValue())
}

func TestAPostOnYouTubeAloneGoesWithNoPhoto(t *testing.T) {
	f := &fake{scheduled: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Scheduled{
		Scheduled: &publishingv1.Scheduled{PostId: "post-2"},
	}}}

	_, err := serve(t, f).Schedule(t.Context(), ringWar(t).At(teasers.Slot{At: at, Platforms: []teasers.Platform{teasers.YouTube}}))

	require.NoError(t, err)
	assert.Empty(t, f.posted.GetTiktokPhotoUrls())
}

func TestWhatPublishingAnswersOfAPostIsADomainError(t *testing.T) {
	for name, answer := range map[string]struct {
		said  *publishingv1.SchedulePostResponse
		check func(t *testing.T, err error)
	}{
		"busy": {
			said: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Busy{
				Busy: &publishingv1.Busy{Reason: "zernio answered 429", RetryAfter: durationpb.New(time.Minute)},
			}},
			check: func(t *testing.T, err error) {
				t.Helper()
				var busy *teasers.BusyError
				require.ErrorAs(t, err, &busy)
				assert.Equal(t, &teasers.BusyError{RetryAfter: time.Minute, Reason: "zernio answered 429"}, busy)
			},
		},
		"taken": {
			said: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Taken{Taken: &publishingv1.Taken{Reason: "no longer open"}}},
			check: func(t *testing.T, err error) {
				t.Helper()
				require.ErrorIs(t, err, teasers.ErrTaken)
			},
		},
		"failed": {
			said: &publishingv1.SchedulePostResponse{Outcome: &publishingv1.SchedulePostResponse_Failed{Failed: &publishingv1.Failed{Reason: "the caption is refused"}}},
			check: func(t *testing.T, err error) {
				t.Helper()
				var refused *teasers.RefusedError
				require.ErrorAs(t, err, &refused)
				assert.Equal(t, "the caption is refused", refused.Reason)
			},
		},
		"nothing": {
			said: &publishingv1.SchedulePostResponse{},
			check: func(t *testing.T, err error) {
				t.Helper()
				require.EqualError(t, err, "publishing said nothing of the post")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := serve(t, &fake{scheduled: answer.said}).Schedule(t.Context(), ringWar(t).At(teasers.Slot{At: at, Platforms: both}))
			answer.check(t, err)
		})
	}
}

func TestAPublishingThatCannotAnswerIsBusyAndARefusalPassesAsItsCode(t *testing.T) {
	_, _, err := serve(t, &fake{err: connect.NewError(connect.CodeUnavailable, errors.New("restarting"))}).Slot(t.Context(), both, at, start)
	var busy *teasers.BusyError
	require.ErrorAs(t, err, &busy)
	assert.Contains(t, busy.Reason, "publishing could not plan a post")

	_, err = serve(t, &fake{err: connect.NewError(connect.CodeFailedPrecondition, errors.New("publishing is off"))}).Teased(t.Context(), at)
	require.ErrorContains(t, err, "publishing refused to list the posts")
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "the error net passes it on as it is")
}
