package receive_reply_handler_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/receive_reply_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/mailroomcontroller/receive_reply_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type stubUseCase struct {
	err        error
	recipients []string
	raws       []string
}

func (s *stubUseCase) Execute(_ context.Context, in receive_reply_usecase.In) (notes.NoteID, error) {
	raw, err := io.ReadAll(in.Raw)
	if err != nil {
		return notes.NoteID{}, fmt.Errorf("the body: %w", err)
	}
	s.recipients, s.raws = append(s.recipients, in.Recipient), append(s.raws, string(raw))
	return notes.NoteID{}, s.err
}

func post(t *testing.T, useCase *stubUseCase, authorization string, body string) int {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, receive_reply_handler.Path, strings.NewReader(body))
	req.Header.Set("Authorization", authorization)
	req.Header.Set(receive_reply_handler.RecipientHeader, "support+1@clickplanet.lol")
	res := httptest.NewRecorder()
	receive_reply_handler.New(useCase, "the-secret", 64).ServeHTTP(res, req)
	return res.Code
}

func TestAMailWithTheSecretIsHandedOnWithItsRecipient(t *testing.T) {
	useCase := &stubUseCase{}

	assert.Equal(t, http.StatusAccepted, post(t, useCase, "Bearer the-secret", "Message-ID: <1@x>\r\n\r\nHi"))

	assert.Equal(t, []string{"support+1@clickplanet.lol"}, useCase.recipients)
	assert.Equal(t, []string{"Message-ID: <1@x>\r\n\r\nHi"}, useCase.raws)
}

func TestNoSecretOrAnotherIsRefusedAndReadsNothing(t *testing.T) {
	useCase := &stubUseCase{}

	for _, authorization := range []string{"", "the-secret", "Bearer the-secret ", "Bearer another", "Basic the-secret"} {
		assert.Equal(t, http.StatusUnauthorized, post(t, useCase, authorization, "Hi"), authorization)
	}
	assert.Empty(t, useCase.raws)
}

func TestAMailOverTheCapIsRefused(t *testing.T) {
	assert.Equal(t, http.StatusRequestEntityTooLarge, post(t, &stubUseCase{}, "Bearer the-secret", strings.Repeat("a", 65)))
}

func TestAMailKeptAlreadyIsDoneWithAStrayOneGoesToTheOperatorAndOneNotKeptIsTriedAgain(t *testing.T) {
	assert.Equal(t, http.StatusAccepted,
		post(t, &stubUseCase{err: fmt.Errorf("wrapped: %w", conversations.ErrReceivedAlready)}, "Bearer the-secret", "Hi"))
	for _, stray := range receive_reply_usecase.Stray {
		assert.Equal(t, http.StatusOK, post(t, &stubUseCase{err: fmt.Errorf("wrapped: %w", stray)}, "Bearer the-secret", "Hi"))
	}

	assert.Equal(t, http.StatusInternalServerError, post(t, &stubUseCase{err: assert.AnError}, "Bearer the-secret", "Hi"))
}
