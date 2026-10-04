package marking_record_take_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/record_take_usecase/marking_record_take"
)

type stubUseCase struct {
	err error
}

func (s stubUseCase) Execute(context.Context, standings.Take) error {
	return s.err
}

type recordingBoards struct {
	marked []standings.Take
}

func (r *recordingBoards) MarkTaken(take standings.Take) {
	r.marked = append(r.marked, take)
}

var take = standings.Take{Account: standings.AccountID{15: 1}, Country: "fr"}

func TestAKeptTakeMarksTheBoards(t *testing.T) {
	boards := &recordingBoards{}

	require.NoError(t, marking_record_take.New(stubUseCase{}, boards).Execute(t.Context(), take))

	assert.Equal(t, []standings.Take{take}, boards.marked)
}

func TestATakeThatFailedMarksNothing(t *testing.T) {
	boards := &recordingBoards{}

	err := marking_record_take.New(stubUseCase{err: assert.AnError}, boards).Execute(t.Context(), take)

	require.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, boards.marked)
}
