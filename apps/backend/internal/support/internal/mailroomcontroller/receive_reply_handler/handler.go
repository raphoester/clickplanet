package receive_reply_handler

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/usecases/receive_reply_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

const (
	Path            = "/mailroom/replies"
	RecipientHeader = "X-Envelope-To"
)

type UseCase interface {
	Execute(ctx context.Context, in receive_reply_usecase.In) (notes.NoteID, error)
}

func New(useCase UseCase, secret string, maxBytes int64) http.Handler {
	return &Handler{useCase: useCase, secret: sha256.Sum256([]byte(secret)), maxBytes: maxBytes}
}

// A mail the mailroom Worker received: a 2xx is done with, kept or dropped; anything else, Cloudflare tries again.
type Handler struct {
	useCase  UseCase
	secret   [sha256.Size]byte
	maxBytes int64
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, bearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	sent := sha256.Sum256([]byte(token))
	if !bearer || subtle.ConstantTimeCompare(sent[:], h.secret[:]) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.ContentLength > h.maxBytes {
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
		return
	}

	_, err := h.useCase.Execute(r.Context(), receive_reply_usecase.In{
		Recipient: r.Header.Get(RecipientHeader), Raw: http.MaxBytesReader(w, r.Body, h.maxBytes),
	})

	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
	case err == nil, receive_reply_usecase.IsDropped(err):
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "not kept, send again", http.StatusInternalServerError)
	}
}
