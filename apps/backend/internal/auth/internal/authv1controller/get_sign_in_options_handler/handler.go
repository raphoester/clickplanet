package get_sign_in_options_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/authprovider"
)

type Offer interface {
	Names() []string
}

func New(offer Offer) GetSignInOptionsHandler {
	return GetSignInOptionsHandler{offer: offer}
}

type GetSignInOptionsHandler struct {
	offer Offer
}

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
