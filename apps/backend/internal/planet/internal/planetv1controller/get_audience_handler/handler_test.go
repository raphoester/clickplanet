package get_audience_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_audience_handler"
)

type stubAudience bonuses.Audience

func (s stubAudience) Audience() bonuses.Audience { return bonuses.Audience(s) }

func TestTheAudienceIsTheRegistrys(t *testing.T) {
	res, err := get_audience_handler.New(stubAudience{Streams: 7, Entrants: 5}).GetAudience(t.Context(),
		connect.NewRequest(&planetv1.GetAudienceRequest{}))
	require.NoError(t, err)

	assert.True(t, proto.Equal(&planetv1.GetAudienceResponse{Streams: 7, Entrants: 5}, res.Msg))
}
