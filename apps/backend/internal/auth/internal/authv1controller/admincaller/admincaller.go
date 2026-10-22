package admincaller

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler/operator_query"
)

type Operators interface {
	Operator(ctx context.Context, cookieHeader string) (*authv1.GetOperatorResponse, error)
}

// The operator the admin page's cookie names, for the calls only that session makes: the gate keeps a token out of
// them, and the console listener, which has no cookie, is Unauthenticated.
func AccountOf(ctx context.Context, operators Operators, header http.Header) (accounts.AccountID, error) {
	operator, err := operators.Operator(ctx, header.Get("Cookie"))

	switch {
	case errors.Is(err, operator_query.ErrNoSession):
		return accounts.AccountID{}, connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, operator_query.ErrNotAnOperator):
		return accounts.AccountID{}, connect.NewError(connect.CodePermissionDenied, err)
	case err != nil:
		return accounts.AccountID{}, fmt.Errorf("failed to read the operator: %w", err)
	}

	account, err := accounts.AccountIDOf(operator.GetAccountId())
	if err != nil {
		return accounts.AccountID{}, fmt.Errorf("failed to read the operator's account: %w", err)
	}
	return account, nil
}
