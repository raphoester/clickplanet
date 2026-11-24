package submit_flag_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/flagmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submissionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/flagsv1controller/submit_flag_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/submit_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	out   submissions.Snapshot
	err   error
	asked []submit_flag_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in submit_flag_usecase.In) (submissions.Snapshot, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

type noLinks struct{}

func (noLinks) HeldURL(string, time.Duration) (string, error) { return "", nil }

func submit(t *testing.T, useCase *stubUseCase) (*flagsv1.SubmitFlagResponse, error) {
	t.Helper()
	ctx := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(t.Context(), "00000000-0000-0000-0000-000000000001"))
	res, err := submit_flag_handler.New(useCase, submissionmessage.New(flagmessage.New("https://media.test"), noLinks{})).
		SubmitFlag(ctx, connect.NewRequest(&flagsv1.SubmitFlagRequest{
			Name: "Rennes", UploadId: "u", Pin: &flagsv1.Pin{Latitude: 48.1, Longitude: -1.6}, Flag: "r-0000000001",
		}))
	if err != nil {
		return nil, fmt.Errorf("SubmitFlag failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheCallerAndWhatItTypedAreHandedOnAndTheWaitingSubmissionAnswered(t *testing.T) {
	at := time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{out: submissions.TestWaiting(submissions.SubmissionID{15: 1}, submissions.AccountID{15: 1}, at)}

	answer, err := submit(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, []submit_flag_usecase.In{{
		Account: "00000000-0000-0000-0000-000000000001", Linked: true, Name: "Rennes", Upload: "u", Latitude: 48.1,
		Longitude: -1.6, Flag: "r-0000000001",
	}}, useCase.asked)
	assert.Equal(t, flagsv1.SubmissionState_SUBMISSION_STATE_WAITING, answer.GetSubmission().GetState())
	assert.Nil(t, answer.GetSubmission().GetPicture(), "its picture is not public before its approval")
}

func TestEachRefusalIsAnsweredWithItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		submissions.ErrGuest:        connect.CodePermissionDenied,
		submissions.ErrNotFounder:   connect.CodePermissionDenied,
		submissions.ErrWaiting:      connect.CodeAlreadyExists,
		catalog.ErrInvalidName:      connect.CodeInvalidArgument,
		catalog.ErrInvalidPin:       connect.CodeInvalidArgument,
		catalog.ErrInvalidFlagID:    connect.CodeInvalidArgument,
		pictures.ErrUnknownUpload:   connect.CodeInvalidArgument,
		pictures.ErrShape:           connect.CodeInvalidArgument,
		pictures.ErrMoves:           connect.CodeInvalidArgument,
		pictures.ErrOff:             connect.CodeFailedPrecondition,
		fmt.Errorf("postgres down"): connect.CodeUnknown,
	} {
		_, got := submit(t, &stubUseCase{err: fmt.Errorf("wrapped: %w", err)})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}

func TestASubmissionThatNamesNoPinIsRefused(t *testing.T) {
	useCase := &stubUseCase{}
	ctx := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(t.Context(), "00000000-0000-0000-0000-000000000001"))

	_, err := submit_flag_handler.New(useCase, submissionmessage.New(flagmessage.New("https://media.test"), noLinks{})).
		SubmitFlag(ctx, connect.NewRequest(&flagsv1.SubmitFlagRequest{Name: "Rennes", UploadId: "u"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}
