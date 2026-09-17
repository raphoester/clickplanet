// Package get_sign_in_options_handler serves auth.v1.AuthService/GetSignInOptions.
package get_sign_in_options_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/authprovider"
)

// Offer is the providers this server signs in with, read from its config at boot.
type Offer interface {
	Names() []string
}

func New(offer Offer) GetSignInOptionsHandler {
	return GetSignInOptionsHandler{offer: offer}
}

type GetSignInOptionsHandler struct {
	offer Offer
}

// GetSignInOptions answers an empty list while sign-in is off, never Unimplemented: the question has an answer either way.
func (h GetSignInOptionsHandler) GetSignInOptions(
	_ context.Context,
	_ *connect.Request[authv1.GetSignInOptionsRequest],
) (*connect.Response[authv1.GetSignInOptionsResponse], error) {
	options := &authv1.GetSignInOptionsResponse{}
	for _, name := range h.offer.Names() {
		options.Providers = append(options.Providers, authprovider.ProtoOf(name))
	}

	res := connect.NewResponse(options)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
