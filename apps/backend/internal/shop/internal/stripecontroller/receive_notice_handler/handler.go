package receive_notice_handler

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
)

const (
	Path            = "/stripe/webhook"
	SignatureHeader = "Stripe-Signature"
)

type UseCase interface {
	Execute(ctx context.Context, notice notices.Notice) error
}

func New(reader notices.Reader, useCase UseCase, maxBytes int64) http.Handler {
	return &Handler{reader: reader, useCase: useCase, maxBytes: maxBytes}
}

// Stripe's webhook endpoint: a 2xx is kept, anything else Stripe sends again for 3 days.
type Handler struct {
	reader   notices.Reader
	useCase  UseCase
	maxBytes int64
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.maxBytes))
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
		return
	case err != nil:
		http.Error(w, "not read, send again", http.StatusBadRequest)
		return
	}

	notice, err := h.reader.Read(payload, r.Header.Get(SignatureHeader))
	switch {
	case errors.Is(err, notices.ErrNotStripes):
		http.Error(w, "not signed by Stripe", http.StatusBadRequest)
		return
	case err != nil:
		http.Error(w, "not read", http.StatusBadRequest)
		return
	}

	if err := h.useCase.Execute(r.Context(), notice); err != nil {
		http.Error(w, "not kept, send again", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
