package flagmessage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions/flagmessage"
)

var (
	at      = time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	mission = flagmissions.MissionID{15: 7}
)

type unknownOrder struct{ flagmissions.TakeIn }

func TestAKindWithNoMessageIsAnError(t *testing.T) {
	_, err := flagmessage.Done("fr", []flagmissions.Done{{
		Mission: flagmissions.MissionSnapshot{ID: mission, Order: unknownOrder{}, Points: 6}, At: at,
	}})

	assert.ErrorIs(t, err, flagmessage.ErrUnknownKind)
}
