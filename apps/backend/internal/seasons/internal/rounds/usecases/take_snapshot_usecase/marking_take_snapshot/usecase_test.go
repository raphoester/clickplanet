package marking_take_snapshot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_snapshot_usecase/marking_take_snapshot"
)

type stubExecutor struct {
	closed []rounds.Round
	err    error
}

func (s stubExecutor) Execute(context.Context) ([]rounds.Round, error) {
	return s.closed, s.err
}

type countingRace struct {
	marks int
}

func (c *countingRace) MarkCounted() {
	c.marks++
}

func TestEverySnapshotMarksTheRace(t *testing.T) {
	race := &countingRace{}
	closed := []rounds.Round{{Finale: true}}

	got, err := marking_take_snapshot.New(stubExecutor{closed: closed}, race).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, closed, got)
	assert.Equal(t, 1, race.marks)
}

func TestASnapshotThatFailedStillMarksTheRaceForTheRoundsItClosed(t *testing.T) {
	race := &countingRace{}
	failure := errors.New("planet is down")

	_, err := marking_take_snapshot.New(stubExecutor{err: failure}, race).Execute(t.Context())

	require.ErrorIs(t, err, failure)
	assert.Equal(t, 1, race.marks)
}
