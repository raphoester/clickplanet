package answer_note_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/answer_note_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/answer_note_handler"
)

const noteID = "019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b"

type stubUseCase struct {
	err   error
	asked []answer_note_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in answer_note_usecase.In) (time.Time, error) {
	s.asked = append(s.asked, in)
	return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC), s.err
}

func answer(t *testing.T, useCase *stubUseCase, id string) (*supportv1.AnswerNoteResponse, error) {
	t.Helper()

	res, err := answer_note_handler.New(useCase).AnswerNote(t.Context(),
		connect.NewRequest(&supportv1.AnswerNoteRequest{NoteId: id, Text: "Fixed."}))
	if err != nil {
		return nil, fmt.Errorf("AnswerNote failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheNoteAndTheTextAreHandedOnAndTheTimeComesBack(t *testing.T) {
	useCase := &stubUseCase{}

	res, err := answer(t, useCase, noteID)

	require.NoError(t, err)
	note, err := notes.NoteIDOf(noteID)
	require.NoError(t, err)
	assert.Equal(t, []answer_note_usecase.In{{Note: note, Text: "Fixed."}}, useCase.asked)
	assert.Equal(t, "2026-11-23T12:00:00Z", res.GetSentAt().AsTime().Format(time.RFC3339))
}

func TestEachRefusalIsTheCodeThatSaysWhatToDo(t *testing.T) {
	for want, err := range map[connect.Code]error{
		connect.CodeNotFound:           notes.ErrUnknownNote,
		connect.CodeFailedPrecondition: notes.ErrNoReplyAddress,
		connect.CodeInvalidArgument:    notes.ErrInvalidText,
		connect.CodeUnavailable:        notes.ErrNotSent,
	} {
		_, got := answer(t, &stubUseCase{err: fmt.Errorf("wrapped: %w", err)}, noteID)

		assert.Equal(t, want, connect.CodeOf(got), "%v", err)
	}

	_, got := answer(t, &stubUseCase{}, "not-a-note")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(got))
}
