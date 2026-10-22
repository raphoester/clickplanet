package authorize_handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/authorize_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in authorize_usecase.In) (string, error)
}

func New(useCase UseCase, logger *slog.Logger) http.Handler {
	return handler{useCase: useCase, logger: logger}
}

type handler struct {
	useCase UseCase
	logger  *slog.Logger
}

// The operator's browser arrives here from the app, and leaves for the admin page, where the operator says yes or no.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	location, err := h.useCase.Execute(r.Context(), authorize_usecase.In{
		ClientID: query.Get("client_id"),
		Ask: connections.Ask{
			ResponseType:    query.Get("response_type"),
			RedirectURI:     query.Get("redirect_uri"),
			Challenge:       query.Get("code_challenge"),
			ChallengeMethod: query.Get("code_challenge_method"),
			State:           query.Get("state"),
		},
	})

	var refusal *connections.Refusal
	switch {
	case err == nil:
		http.Redirect(w, r, location, http.StatusFound)
	case errors.As(err, &refusal):
		http.Redirect(w, r, refusal.URL, http.StatusFound)
	case errors.Is(err, connections.ErrInvalidClient), errors.Is(err, connections.ErrInvalidRedirect):
		// RFC 6749 4.1.2.1: never redirect to a place nobody vouched for.
		http.Error(w, "This app is not registered with ClickPlanet: connect it again from the app.", http.StatusBadRequest)
	default:
		h.logger.Error("failed to authorize an app", slog.Any("error", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
