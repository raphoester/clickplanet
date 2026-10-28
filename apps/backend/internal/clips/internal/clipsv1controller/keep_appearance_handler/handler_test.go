package keep_appearance_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/keep_appearance_handler"
)

type useCase struct {
	err      error
	cast     casts.CastID
	fighters []casts.Number
	post     appearances.Post
}

func (u *useCase) Execute(_ context.Context, cast casts.CastID, fighters []casts.Number, post appearances.Post) (int, error) {
	u.cast, u.fighters, u.post = cast, fighters, post
	return len(fighters), u.err
}

const castID = "5f0c2a11-0b6d-4f7e-9a51-3f1f8f0c2a11"

var scheduledAt = time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)

func request() *clipsv1.KeepAppearanceRequest {
	return &clipsv1.KeepAppearanceRequest{
		CastId: castID, Fighters: []uint32{2, 1}, PostId: "p1", Title: "FRANCE VS TURKEY",
		Platforms: []string{"tiktok", "youtube"}, ScheduledAt: timestamppb.New(scheduledAt),
	}
}

func TestTheRequestIsHandedOverAsTheDomainSaysIt(t *testing.T) {
	inner := &useCase{}

	answer, err := keep_appearance_handler.New(inner).KeepAppearance(t.Context(), connect.NewRequest(request()))
	require.NoError(t, err)

	assert.Equal(t, uint32(2), answer.Msg.GetKept())
	assert.Equal(t, castID, inner.cast.String())
	assert.Equal(t, []casts.Number{2, 1}, inner.fighters)
	assert.Equal(t, appearances.Post{
		ID: "p1", Title: "FRANCE VS TURKEY",
		Platforms: []appearances.Platform{appearances.TikTok, appearances.YouTube}, ScheduledAt: scheduledAt,
	}, inner.post)
}

func TestWhatTheCallerGotWrongIsInvalidArgument(t *testing.T) {
	for name, change := range map[string]func(*clipsv1.KeepAppearanceRequest){
		"no cast":     func(r *clipsv1.KeepAppearanceRequest) { r.CastId = "" },
		"no post":     func(r *clipsv1.KeepAppearanceRequest) { r.PostId = "" },
		"no platform": func(r *clipsv1.KeepAppearanceRequest) { r.Platforms = nil },
		"a platform":  func(r *clipsv1.KeepAppearanceRequest) { r.Platforms = []string{"instagram"} },
		"no time":     func(r *clipsv1.KeepAppearanceRequest) { r.ScheduledAt = nil },
	} {
		asked := request()
		change(asked)
		_, err := keep_appearance_handler.New(&useCase{}).KeepAppearance(t.Context(), connect.NewRequest(asked))
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), name)
	}
}

func TestTheUseCasesRefusalsKeepTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		casts.ErrNoCast: connect.CodeNotFound,
		fmt.Errorf("%w: 3", casts.ErrUnknownFighter): connect.CodeInvalidArgument,
		errors.New("postgres is down"):               connect.CodeUnknown,
	} {
		_, got := keep_appearance_handler.New(&useCase{err: err}).KeepAppearance(t.Context(), connect.NewRequest(request()))
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
