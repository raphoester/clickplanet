package allegiance_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_allegiance_handler/allegiance_query"
)

const alice allegiances.Account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

type stubAllegiances map[allegiances.Account]allegiances.Snapshot

func (s stubAllegiances) Snapshot(account allegiances.Account) allegiances.Snapshot {
	return s[account]
}

func TestAnAllegianceSaysWhenItMayChange(t *testing.T) {
	changed := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	query := allegiance_query.NewMemoryQuery(stubAllegiances{alice: {Flag: "de", ChangedAt: changed}})

	assert.True(t, proto.Equal(&planetv1.GetAllegianceResponse{Allegiance: &planetv1.Allegiance{
		CountryId: "de", ChangeableAtUnixMs: changed.Add(time.Hour).UnixMilli(),
	}}, query.Allegiance(alice)))
}

func TestAFirstFlagMayChangeNow(t *testing.T) {
	query := allegiance_query.NewMemoryQuery(stubAllegiances{alice: {Flag: "fr"}})

	assert.True(t, proto.Equal(&planetv1.GetAllegianceResponse{Allegiance: &planetv1.Allegiance{CountryId: "fr"}},
		query.Allegiance(alice)))
}

func TestNoFlagYetAndNoAccountAnswerNoAllegiance(t *testing.T) {
	query := allegiance_query.NewMemoryQuery(stubAllegiances{alice: {Flag: "fr"}})

	assert.Nil(t, query.Allegiance("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e60").GetAllegiance())
	assert.Nil(t, query.Allegiance(allegiances.NoAccount).GetAllegiance())
}
