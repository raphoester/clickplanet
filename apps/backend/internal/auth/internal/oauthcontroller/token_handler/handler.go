package token_handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/exchange_code_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/refresh_tokens_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/oauthcontroller"
)

type Exchange interface {
	Execute(ctx context.Context, in exchange_code_usecase.In) (*admintokens.Pair, error)
}

type Refresh interface {
	Execute(ctx context.Context, in refresh_tokens_usecase.In) (*admintokens.Pair, error)
}

func New(exchange Exchange, refresh Refresh, logger *slog.Logger) http.Handler {
	return handler{exchange: exchange, refresh: refresh, logger: logger}
}

type handler struct {
	exchange Exchange
	refresh  Refresh
	logger   *slog.Logger
}

type tokens struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, oauthcontroller.MaxBody)
	if err := r.ParseForm(); err != nil {
		oauthcontroller.Error(w, http.StatusBadRequest, "invalid_request", "the body is not a form")
		return
	}

	// A public client sends its id in the form; one that sends it as HTTP Basic, with any secret, is read the same.
	clientID := r.PostForm.Get("client_id")
	if basic, _, ok := r.BasicAuth(); ok && clientID == "" {
		clientID = basic
	}

	var (
		pair *admintokens.Pair
		err  error
	)
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		pair, err = h.exchange.Execute(r.Context(), exchange_code_usecase.In{
			Code: r.PostForm.Get("code"),
			Redemption: connections.Redemption{
				ClientID: clientID, RedirectURI: r.PostForm.Get("redirect_uri"), Verifier: r.PostForm.Get("code_verifier"),
			},
		})
	case "refresh_token":
		pair, err = h.refresh.Execute(r.Context(), refresh_tokens_usecase.In{
			RefreshToken: admintokens.Secret(r.PostForm.Get("refresh_token")), ClientID: clientID,
		})
	default:
		oauthcontroller.Error(w, http.StatusBadRequest, "unsupported_grant_type", "a code or a refresh token")
		return
	}

	switch {
	case errors.Is(err, connections.ErrInvalidGrant), errors.Is(err, admintokens.ErrUnknownToken):
		oauthcontroller.Error(w, http.StatusBadRequest, "invalid_grant", "connect the app again")
		return
	case err != nil:
		h.logger.Error("failed to issue admin tokens", slog.Any("error", err))
		oauthcontroller.Error(w, http.StatusInternalServerError, "server_error", "internal error")
		return
	}

	oauthcontroller.JSON(w, http.StatusOK, tokens{
		AccessToken:  string(pair.Access()),
		TokenType:    "Bearer",
		ExpiresIn:    int(admintokens.AccessLifetime.Seconds()),
		RefreshToken: string(pair.Refresh()),
	})
}
