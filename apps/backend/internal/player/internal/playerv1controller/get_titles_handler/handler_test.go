package get_titles_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/get_titles_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	dashboard get_titles_usecase.Dashboard
}

func (s stubUseCase) Execute(context.Context, players.AccountID) (get_titles_usecase.Dashboard, error) {
	return s.dashboard, nil
}

var conquest = titles.Place{Track: "conquest", TrackName: "Conquest", Number: 1, Count: 5}

func TestTheDashboardIsMapped(t *testing.T) {
	settler := titles.Standing{Title: titles.Settler{}, Place: conquest}
	useCase := stubUseCase{dashboard: get_titles_usecase.Dashboard{
		Showcase: wearing.Showcase{Worn: settler, Shown: []titles.Standing{{Title: titles.OG{}}, settler}},
		Tracks: []titles.TrackProgress{{ID: "conquest", Name: "Conquest", Progress: 150, Steps: []titles.Step{
			{Standing: settler, Threshold: 100, Earned: true},
		}}},
	}}
	ctx := cpctx.AddAccountToContext(t.Context(), players.AccountID{15: 1}.String())

	res, err := get_titles_handler.New(useCase).GetTitles(ctx, connect.NewRequest(&playerv1.GetTitlesRequest{}))

	require.NoError(t, err)
	assert.Equal(t, "settler", res.Msg.GetWorn().GetId())
	require.Len(t, res.Msg.GetWearable(), 2)
	assert.Nil(t, res.Msg.GetWearable()[0].GetRank())
	require.Len(t, res.Msg.GetTracks(), 1)
	track := res.Msg.GetTracks()[0]
	assert.Equal(t, uint64(150), track.GetProgress())
	assert.Equal(t, uint64(100), track.GetSteps()[0].GetThreshold())
	assert.True(t, track.GetSteps()[0].GetEarned())
	assert.Equal(t, uint32(1), track.GetSteps()[0].GetTitle().GetRank().GetNumber())
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	_, err := get_titles_handler.New(stubUseCase{}).GetTitles(t.Context(), connect.NewRequest(&playerv1.GetTitlesRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
