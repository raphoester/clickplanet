package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestAReaderReadsItsPartOfTheSetEnforcedLast(t *testing.T) {
	inForce := cprules.NewInForce[*planetv1.Rules](cptime.SystemClock{})
	throttle := cprules.Reader(inForce, clicks.ThrottlePart)
	toll := cprules.Reader(inForce, clicks.TollPart)
	wallet := cprules.Reader(inForce, charges.Part)

	for _, kept := range []cprules.Snapshot[*planetv1.Rules]{rules.Seeded(), rush()} {
		inForce.Enforce(restored(t, kept))
	}

	assert.Equal(t, cprules.SetName("rush"), inForce.Name())
	assert.InDelta(t, 3.0, throttle().Tempo, 1e-9)
	assert.Len(t, toll(), 3)
	assert.Equal(t, 30, wallet().Spread.Clicks)
}
