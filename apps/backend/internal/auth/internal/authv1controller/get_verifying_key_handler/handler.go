// Package get_verifying_key_handler serves auth.v1.InternalService/GetVerifyingKey.
package get_verifying_key_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
)

// Keys is the signer, asked only for the half that is not secret.
type Keys interface {
	PublicKey() string
}

func New(keys Keys) GetVerifyingKeyHandler {
	return GetVerifyingKeyHandler{keys: keys}
}

type GetVerifyingKeyHandler struct {
	keys Keys
}

// GetVerifyingKey answers no-store: a caller keeps the key for its own lifetime, not a proxy's.
func (h GetVerifyingKeyHandler) GetVerifyingKey(
	_ context.Context,
	_ *connect.Request[authv1.GetVerifyingKeyRequest],
) (*connect.Response[authv1.GetVerifyingKeyResponse], error) {
	res := connect.NewResponse(&authv1.GetVerifyingKeyResponse{PublicKey: h.keys.PublicKey()})
	res.Header().Set("Cache-Control", "no-store")

	return res, nil
}
