package get_landmass_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_landmass_handler"
)

type stubQuery struct {
	answer *planetv1.GetLandmassResponse
	asked  []clicks.LandmassID
}

func (s *stubQuery) Landmass(_ context.Context, landmass clicks.LandmassID) (*planetv1.GetLandmassResponse, error) {
	s.asked = append(s.asked, landmass)
	return s.answer, nil
}

func ask(t *testing.T, query *stubQuery, landmass uint32) (*connect.Response[planetv1.GetLandmassResponse], error) {
	t.Helper()

	borders := clicks.BordersOf(10, []uint32{1, 2}, []uint32{3})
	return get_landmass_handler.New(borders, query).GetLandmass(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&planetv1.GetLandmassRequest{LandmassId: landmass}))
}

func TestTheLandmassIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetLandmassResponse{
		Holders: []*planetv1.LandmassHolder{{AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", CountryId: "fr", Tiles: 2}},
	}}

	res, err := ask(t, query, 2)

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, []clicks.LandmassID{2}, query.asked)
}

func TestALandmassOffTheBordersIsInvalidAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	for _, landmass := range []uint32{0, 3, 70_000} {
		_, err := ask(t, query, landmass)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "landmass %d", landmass)
	}
	assert.Empty(t, query.asked)
}
