package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestAReaderReadsItsPartOfTheSetEnforcedLast(t *testing.T) {
	inForce := cprules.NewInForce[*boxesv1.Rules](cptime.SystemClock{})
	read := cprules.Reader(inForce, boxes.Part)

	for _, kept := range []cprules.Snapshot[*boxesv1.Rules]{rules.Seeded(), rush()} {
		inForce.Enforce(restored(t, kept))
	}

	assert.Equal(t, cprules.SetName("rush"), inForce.Name())
	assert.Equal(t, 30, read().Pace.MaxBoxesPerHour)
	assert.Equal(t, time.Minute, read().Pace.MinInterval)
}
