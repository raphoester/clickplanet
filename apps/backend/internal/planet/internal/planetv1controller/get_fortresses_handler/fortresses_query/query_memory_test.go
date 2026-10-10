package fortresses_query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_fortresses_handler/fortresses_query"
)

type stubFortresses map[clicks.LandmassID]string

func (s stubFortresses) Fortresses() map[clicks.LandmassID]string { return s }

func TestEveryLockedLandmassIsListedInOrder(t *testing.T) {
	answer := fortresses_query.NewMemoryQuery(stubFortresses{257: "bg", 4: "it"}).Fortresses()

	assert.True(t, proto.Equal(&planetv1.GetFortressesResponse{Fortresses: []*planetv1.Fortress{
		{LandmassId: 4, CountryId: "it"},
		{LandmassId: 257, CountryId: "bg"},
	}}, answer))
}
