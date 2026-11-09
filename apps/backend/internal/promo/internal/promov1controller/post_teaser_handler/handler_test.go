package post_teaser_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/post_teaser_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

type useCase struct {
	outcome teasers.Outcome
	err     error
	battle  teasers.BattleID
	media   teasers.Media
}

func (u *useCase) Execute(_ context.Context, battle teasers.BattleID, media teasers.Media) (teasers.Outcome, error) {
	u.battle, u.media = battle, media
	return u.outcome, u.err
}

func post(t *testing.T, u *useCase, req *promov1.PostTeaserRequest) (*connect.Response[promov1.PostTeaserResponse], error) {
	t.Helper()
	return post_teaser_handler.New(u).PostTeaser(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the test reads it.
}

var request = &promov1.PostTeaserRequest{
	BattleId: "entry-7", PhotoUrls: []string{"https://media.example/card-1.jpg"}, VideoUrl: "https://media.example/teaser.mp4",
}

func TestTheBattleAndItsMediaGoToTheUseCase(t *testing.T) {
	u := &useCase{outcome: teasers.Outcome{Result: teasers.Held, Why: "no room"}}

	_, err := post(t, u, request)

	require.NoError(t, err)
	assert.Equal(t, teasers.BattleID("entry-7"), u.battle)
	assert.Equal(t, teasers.Media{Photos: []teasers.MediaURL{"https://media.example/card-1.jpg"}, Video: "https://media.example/teaser.mp4"}, u.media)
}

func TestEachOutcomeIsAnsweredAsItIs(t *testing.T) {
	at := time.Date(2026, 11, 13, 14, 0, 0, 0, time.UTC)
	for _, each := range []struct {
		outcome teasers.Outcome
		answer  *promov1.PostTeaserResponse
	}{
		{
			outcome: teasers.Outcome{
				Result: teasers.Posted, Title: "BATTLE", At: at, Post: "post-1", Platforms: []teasers.Platform{teasers.TikTok, teasers.YouTube},
				Refused: map[teasers.Platform]string{teasers.YouTube: "no account"},
			},
			answer: &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Posted{Posted: &promov1.TeaserPosted{
				PostId: "post-1", Title: "BATTLE", At: timestamppb.New(at), Platforms: []string{"tiktok", "youtube"},
				Refusals: []*promov1.TeaserRefusal{{Platform: "youtube", Reason: "no account"}},
			}}},
		},
		{
			outcome: teasers.Outcome{Result: teasers.Held, Why: "no room"},
			answer:  &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Held{Held: &promov1.TeaserNotPosted{Reason: "no room"}}},
		},
		{
			outcome: teasers.Outcome{Result: teasers.Skipped, Why: "teased already"},
			answer:  &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Skipped{Skipped: &promov1.TeaserNotPosted{Reason: "teased already"}}},
		},
		{
			outcome: teasers.Outcome{Result: teasers.Busy, Why: "429", RetryAfter: time.Minute},
			answer: &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Busy{Busy: &promov1.TeaserBusy{
				Reason: "429", RetryAfter: durationpb.New(time.Minute),
			}}},
		},
		{
			outcome: teasers.Outcome{Result: teasers.Busy, Why: "restarting"},
			answer:  &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Busy{Busy: &promov1.TeaserBusy{Reason: "restarting"}}},
		},
		{
			outcome: teasers.Outcome{Result: teasers.Failed, Why: "refused"},
			answer:  &promov1.PostTeaserResponse{Outcome: &promov1.PostTeaserResponse_Failed{Failed: &promov1.TeaserNotPosted{Reason: "refused"}}},
		},
	} {
		answer, err := post(t, &useCase{outcome: each.outcome}, request)

		require.NoError(t, err)
		assert.True(t, proto.Equal(each.answer, answer.Msg), "%s: %v", each.outcome.Result, answer.Msg)
	}
}

func TestARequestWithNoBattleOrMediaNotTheBattlesIsInvalid(t *testing.T) {
	u := &useCase{}
	_, err := post(t, u, &promov1.PostTeaserRequest{VideoUrl: "https://media.example/teaser.mp4"})
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, u.battle, "the use case is not asked")

	_, err = post(t, &useCase{err: fmt.Errorf("%w: no video", teasers.ErrInvalidPost)}, request)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	off := connect.NewError(connect.CodeFailedPrecondition, errors.New("publishing is off"))
	_, err = post(t, &useCase{err: off}, request)
	require.ErrorIs(t, err, off)
}
