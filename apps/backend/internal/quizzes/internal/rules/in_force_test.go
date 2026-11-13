package rules_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	quizzesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/quizzes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestAReaderReadsItsPartOfTheSetEnforcedLast(t *testing.T) {
	inForce := cprules.NewInForce[*quizzesv1.Rules](cptime.SystemClock{})
	asked := cprules.Reader(inForce, quizzes.Part)

	for _, kept := range []cprules.Snapshot[*quizzesv1.Rules]{rules.Seeded(), rush()} {
		inForce.Enforce(restored(t, kept))
	}

	assert.Equal(t, cprules.SetName("rush"), inForce.Name())
	assert.Equal(t, 30, asked().MaxRightAnswersPerHour)
	assert.Equal(t, time.Minute, asked().MinInterval)
}
