package approve_submission_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/approve_submission_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/approve_submission_usecase"
)

type stubUseCase struct {
	asked []approve_submission_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in approve_submission_usecase.In) (submissions.Snapshot, error) {
	s.asked = append(s.asked, in)
	approved := submissions.TestWaiting(submissions.SubmissionID{15: 1}, catalog.AccountID{15: 1}, time.Now())
	approved.State, approved.Held = submissions.Approved, false
	return approved, nil
}

type noLinks struct{}

func (noLinks) HeldURL(string, time.Duration) (string, error) { return "", nil }

func approve(t *testing.T, useCase *stubUseCase, msg *flagsv1.ApproveSubmissionRequest) {
	t.Helper()
	_, err := approve_submission_handler.New(useCase, submissionmessage.New(flagmessage.New("https://media.test"), noLinks{})).
		ApproveSubmission(t.Context(), connect.NewRequest(msg))
	require.NoError(t, err)
}

func TestNoCapitalInTheRequestKeepsThePlayersAndOneGivenMovesIt(t *testing.T) {
	useCase := &stubUseCase{}

	approve(t, useCase, &flagsv1.ApproveSubmissionRequest{SubmissionId: "s"})
	approve(t, useCase, &flagsv1.ApproveSubmissionRequest{
		SubmissionId: "s", Pin: &flagsv1.Pin{Latitude: 47.218, Longitude: -1.5536},
	})

	assert.Equal(t, []approve_submission_usecase.In{
		{Submission: "s"},
		{Submission: "s", Moved: true, Latitude: 47.218, Longitude: -1.5536},
	}, useCase.asked)
}
