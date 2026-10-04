package marking_forget_account_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/forget_account_usecase/marking_forget_account"
)

type stubUseCase struct {
	err error
}

func (s stubUseCase) Execute(context.Context, standings.AccountID) error {
	return s.err
}

type recordingBoards struct {
	marked []standings.AccountID
}

func (r *recordingBoards) MarkForgotten(account standings.AccountID) {
	r.marked = append(r.marked, account)
}

var ana = standings.AccountID{15: 1}

func TestAForgottenAccountMarksTheBoards(t *testing.T) {
	boards := &recordingBoards{}

	require.NoError(t, marking_forget_account.New(stubUseCase{}, boards).Execute(t.Context(), ana))

	assert.Equal(t, []standings.AccountID{ana}, boards.marked)
}

func TestAFailureToForgetMarksNothing(t *testing.T) {
	boards := &recordingBoards{}

	err := marking_forget_account.New(stubUseCase{err: assert.AnError}, boards).Execute(t.Context(), ana)

	require.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, boards.marked)
}
