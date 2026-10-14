package create_upload_handler_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/create_upload_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/create_upload_handler"
)

type stubUseCase struct {
	out   create_upload_usecase.Out
	err   error
	asked []create_upload_usecase.In
}

func (s *stubUseCase) Execute(in create_upload_usecase.In) (create_upload_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func create(t *testing.T, useCase *stubUseCase, msg *supportv1.CreateUploadRequest) (*supportv1.CreateUploadResponse, error) {
	t.Helper()

	res, err := create_upload_handler.New(useCase).CreateUpload(t.Context(), connect.NewRequest(msg))
	if err != nil {
		return nil, fmt.Errorf("CreateUpload failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheTypeAndTheSizeAreHandedOnAndTheIDAndTheLinkAnswered(t *testing.T) {
	useCase := &stubUseCase{out: create_upload_usecase.Out{ID: notes.AttachmentID{15: 1}, URL: "https://bucket.example/a"}}

	answer, err := create(t, useCase, &supportv1.CreateUploadRequest{ContentType: "video/mp4", SizeBytes: 9000})

	require.NoError(t, err)
	assert.Equal(t, []create_upload_usecase.In{{MediaType: notes.MediaMP4, Size: 9000}}, useCase.asked)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", answer.GetAttachmentId())
	assert.Equal(t, "https://bucket.example/a", answer.GetUrl())
}

func TestASizeNoFileHasStaysTooBigAndNeverTurnsNegative(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := create(t, useCase, &supportv1.CreateUploadRequest{ContentType: "video/mp4", SizeBytes: math.MaxUint64})

	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64), useCase.asked[0].Size)
}

func TestAFileTheRulesDoNotTakeIsTheCallersFault(t *testing.T) {
	_, err := create(t, &stubUseCase{err: fmt.Errorf("wrapped: %w", notes.ErrInvalidAttachment)},
		&supportv1.CreateUploadRequest{ContentType: "text/html", SizeBytes: 10})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestALinkThatCannotBeMadeIsLeftToTheErrorNet(t *testing.T) {
	_, err := create(t, &stubUseCase{err: errors.New("no bucket")},
		&supportv1.CreateUploadRequest{ContentType: "image/png", SizeBytes: 10})

	assert.ErrorContains(t, err, "no bucket")
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
