package roster_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler/roster_query"
)

type stubLines []*playerv1.RosterEntry

func (s stubLines) Lines() []*playerv1.RosterEntry {
	return s
}

func TestTheRosterIsTheLinesInTheirOrder(t *testing.T) {
	lines := stubLines{{Key: "2", Name: "Ada_L", CountryId: "fr"}, {Key: "1", Name: "guest_0b1c2d", Guest: true}}

	assert.True(t, proto.Equal(
		&playerv1.GetRosterResponse{Entries: lines},
		roster_query.NewMemoryQuery(lines).Roster(),
	))
}
