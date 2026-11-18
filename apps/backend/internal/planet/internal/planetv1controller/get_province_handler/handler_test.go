package get_province_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_province_handler"
)

type stubQuery struct {
	answer *planetv1.GetProvinceResponse
	asked  []clicks.ProvinceID
}

func (s *stubQuery) Province(_ context.Context, province clicks.ProvinceID) (*planetv1.GetProvinceResponse, error) {
	s.asked = append(s.asked, province)
	return s.answer, nil
}

// board has provinces 1 and 2.
type board struct{}

func (board) Provinces() *clicks.Provinces {
	return clicks.ProvincesOf("test-provinces", clicks.BordersOf(10, []uint32{1, 2}, []uint32{3}), []uint32{1, 2}, []uint32{3})
}

func ask(t *testing.T, query *stubQuery, province uint32) (*connect.Response[planetv1.GetProvinceResponse], error) {
	t.Helper()

	return get_province_handler.New(board{}, query).GetProvince(t.Context(), //nolint:wrapcheck // the test reads the handler's own error.
		connect.NewRequest(&planetv1.GetProvinceRequest{ProvinceId: province}))
}

func TestTheProvinceIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &planetv1.GetProvinceResponse{
		LastFortification: &planetv1.ProvinceFortification{CountryId: "fr", FortifiedAtUnixMs: 1},
	}}

	res, err := ask(t, query, 2)

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, []clicks.ProvinceID{2}, query.asked)
}

func TestAProvinceOffTheFileIsInvalidAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	for _, province := range []uint32{0, 3, 70_000} {
		_, err := ask(t, query, province)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "province %d", province)
	}
	assert.Empty(t, query.asked)
}
