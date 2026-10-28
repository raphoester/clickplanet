package make_cast_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/make_cast_handler"
)

var now = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

type useCase struct {
	cast  casts.Cast
	takes []casts.NumberedTake
}

func (u useCase) Execute(_ context.Context, since, until time.Time) (casts.Cast, []casts.NumberedTake, error) {
	if _, err := casts.WindowOf(since, until, now); err != nil {
		return casts.Cast{}, nil, err //nolint:wrapcheck // the fake answers as the use case does.
	}
	return u.cast, u.takes, nil
}

func TestTheCastAnswersNamesAndNumbersAndNoAccount(t *testing.T) {
	id, err := casts.CastIDOf("5f0c2a11-0b6d-4f7e-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	ada, err := casts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a01")
	require.NoError(t, err)
	window := casts.Window{Since: now.Add(-12 * time.Hour), Until: now}
	inner := useCase{
		cast:  casts.CastSnapshot{ID: id, MadeAt: now, Window: window, Fighters: []casts.FighterSnapshot{{Number: 1, Account: ada, Name: "Ada"}}}.Restore(),
		takes: []casts.NumberedTake{{At: now.Add(-time.Hour), Tile: 7, Country: "fr", Previous: "de", Fighter: 1}},
	}

	answer, err := make_cast_handler.New(inner).MakeCast(t.Context(), connect.NewRequest(&clipsv1.MakeCastRequest{
		Since: timestamppb.New(window.Since),
	}))
	require.NoError(t, err)

	want := &clipsv1.Cast{
		Id: id.String(), Since: timestamppb.New(window.Since), Until: timestamppb.New(now),
		Fighters: []*clipsv1.Fighter{{Number: 1, Name: "Ada"}},
		Takes:    []*clipsv1.CastTake{{At: timestamppb.New(now.Add(-time.Hour)), TileId: 7, CountryId: "fr", PreviousCountryId: "de", Fighter: 1}},
	}
	assert.True(t, proto.Equal(want, answer.Msg.GetCast()), "%v", answer.Msg.GetCast())
	assert.NotContains(t, answer.Msg.String(), ada.String())
}

func TestAWindowTheLedgerCannotReadIsInvalidArgument(t *testing.T) {
	_, err := make_cast_handler.New(useCase{}).MakeCast(t.Context(), connect.NewRequest(&clipsv1.MakeCastRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
