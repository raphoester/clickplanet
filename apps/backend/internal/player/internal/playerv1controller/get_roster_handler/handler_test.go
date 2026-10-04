package get_roster_handler_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler"
)

type stubQuery struct {
	answer *playerv1.GetRosterResponse
}

func (s stubQuery) Roster() *playerv1.GetRosterResponse {
	return s.answer
}

func TestTheRosterIsTheQuerysAndMayBeCached(t *testing.T) {
	query := stubQuery{answer: &playerv1.GetRosterResponse{Entries: []*playerv1.RosterEntry{{Key: "k1", Name: "Ada_L", CountryId: "fr"}}}}

	res, err := get_roster_handler.New(query).GetRoster(t.Context(), connect.NewRequest(&playerv1.GetRosterRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	assert.Equal(t, "public, max-age=5", res.Header().Get("Cache-Control"))
}
