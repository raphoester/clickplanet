package allegiance_facts_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances/allegiance_facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestAPledgeIsToldWithTheFlagItLeft(t *testing.T) {
	at := time.Date(2026, 11, 2, 20, 0, 0, 0, time.UTC)
	log := facts.NewLog(cptime.NewFixedClock(at))

	allegiance_facts.New(log).Pledged("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f", "de", "fr")

	told := log.Take()
	require.Len(t, told, 1)
	want := &planetv1.AllegiancePledged{
		AccountId: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f", Country: "de", Previous: "fr", PledgedAt: timestamppb.New(at),
	}
	assert.True(t, proto.Equal(want, told[0]), "got %v", told[0])
}
