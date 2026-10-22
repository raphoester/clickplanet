package register_handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/register_client_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/oauthcontroller"
)

type UseCase interface {
	Execute(ctx context.Context, in register_client_usecase.In) (register_client_usecase.Out, error)
}

func New(useCase UseCase, logger *slog.Logger) http.Handler {
	return handler{useCase: useCase, logger: logger}
}

type handler struct {
	useCase UseCase
	logger  *slog.Logger
}

// RFC 7591's request: what an app asks for beyond its name and its redirects is answered with what it gets.
type request struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

type response struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var asked request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, oauthcontroller.MaxBody)).Decode(&asked); err != nil {
		oauthcontroller.Error(w, http.StatusBadRequest, "invalid_client_metadata", "the body is not a JSON registration")
		return
	}

	out, err := h.useCase.Execute(r.Context(), register_client_usecase.In{Name: asked.ClientName, RedirectURIs: asked.RedirectURIs})
	switch {
	case errors.Is(err, connections.ErrInvalidRedirect):
		oauthcontroller.Error(w, http.StatusBadRequest, "invalid_redirect_uri", "only the redirects this server knows may be registered")
		return
	case err != nil:
		h.logger.Error("failed to register a client", slog.Any("error", err))
		oauthcontroller.Error(w, http.StatusInternalServerError, "server_error", "internal error")
		return
	}

	oauthcontroller.JSON(w, http.StatusCreated, response{
		ClientID:                out.ClientID,
		ClientIDIssuedAt:        out.Client.IssuedAt.Unix(),
		ClientName:              out.Client.Name,
		RedirectURIs:            out.Client.RedirectURIs,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	})
}
