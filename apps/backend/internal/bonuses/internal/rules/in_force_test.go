package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestAReaderReadsItsPartOfTheSetEnforcedLast(t *testing.T) {
	inForce := cprules.NewInForce[*bonusesv1.Rules](cptime.SystemClock{})
	offers := cprules.Reader(inForce, bonuses.Part)

	for _, kept := range []cprules.Snapshot[*bonusesv1.Rules]{rules.Seeded(), rush()} {
		inForce.Enforce(restored(t, kept))
	}

	assert.Equal(t, cprules.SetName("rush"), inForce.Name())
	assert.Equal(t, 30, offers().Boxes.MaxChargesPerHour)
	assert.Equal(t, time.Minute, offers().Boxes.MinInterval)
}
