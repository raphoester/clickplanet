package answer_appeal_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals/usecases/answer_appeal_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var (
	askedAt = time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	smurfs  = submissions.SubmissionID{15: 1}
)

func TestTheOperatorsVerdictAnswersTheAppealOnce(t *testing.T) {
	store := appeals.NewMemoryStore()
	appeal, err := appeals.Snapshot{
		Submission: smurfs, Account: catalog.AccountID{15: 1}, State: appeals.Asked, AskedAt: askedAt,
		DecidedAt: askedAt, Rejection: submissions.Rejection{Reason: submissions.NotAPlace},
	}.Restore()
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), appeal))
	useCase := answer_appeal_usecase.New(cpatomic.FakeTransactioner{}, store)

	require.NoError(t, useCase.Execute(t.Context(), smurfs.String(), askedAt.Add(-time.Hour)))
	assert.Equal(t, appeals.Asked, store.Kept()[smurfs].State, "the judge's rejection, heard after the appeal")

	require.NoError(t, useCase.Execute(t.Context(), smurfs.String(), askedAt.Add(time.Hour)))
	require.NoError(t, useCase.Execute(t.Context(), smurfs.String(), askedAt.Add(2*time.Hour)))
	assert.Equal(t, appeals.Answered, store.Kept()[smurfs].State)
	assert.Equal(t, askedAt.Add(time.Hour), store.Kept()[smurfs].DecidedAt, "told again, nothing moves")

	require.NoError(t, useCase.Execute(t.Context(), submissions.SubmissionID{15: 2}.String(), askedAt),
		"a submission never appealed")
}
